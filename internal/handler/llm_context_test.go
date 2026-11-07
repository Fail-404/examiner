package handler

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/pavelanni/examiner/internal/llm"
	"github.com/pavelanni/examiner/internal/model"
)

type requestMarker struct{}

// contextEvaluator наблюдает контекст на границе обработчика и клиента LLM.
type contextEvaluator struct {
	evaluate func(context.Context) error
	grade    func(context.Context)
}

func (e contextEvaluator) EvaluateAnswer(ctx context.Context, _ model.Question, _ []model.Message, _ int, _, _ int64) (*llm.GradeResult, string, error) {
	if err := e.evaluate(ctx); err != nil {
		return nil, "", err
	}
	return &llm.GradeResult{Score: 5, MaxPoints: 10}, "", nil
}

func (e contextEvaluator) GradeThread(ctx context.Context, _ model.Question, _ []model.Message, _, _ int64) (*llm.GradeResult, error) {
	e.grade(ctx)
	return &llm.GradeResult{Score: 5, MaxPoints: 10}, nil
}

// TestAnswerCancelsLLMRequest проверяет отмену во время оценки и сохранность ответа.
func TestAnswerCancelsLLMRequest(t *testing.T) {
	entered := make(chan struct{})
	testCtx, stop := context.WithTimeout(t.Context(), 2*time.Second)
	defer stop()
	ev := contextEvaluator{evaluate: func(ctx context.Context) error {
		if ctx.Value(requestMarker{}) != "answer" {
			t.Error("request values were not propagated")
		}
		close(entered)
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-testCtx.Done():
			return testCtx.Err()
		}
	}}
	env := newAnswerEnv(t, 3, ev)
	ctx, cancel := context.WithCancel(context.WithValue(testCtx, requestMarker{}, "answer"))
	defer cancel()
	form := url.Values{"answer": {"saved answer"}}
	req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/exam/%d/answer/%d", env.sessID, env.threadID), strings.NewReader(form.Encode())).WithContext(ctx)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	finished := make(chan struct{})
	go func() { env.router.ServeHTTP(httptest.NewRecorder(), req); close(finished) }()
	select {
	case <-entered:
	case <-testCtx.Done():
		t.Fatal("evaluator was not called")
	}
	cancel()
	select {
	case <-finished:
	case <-testCtx.Done():
		t.Fatal("LLM ignored request cancellation")
	}
	if got := env.messageCount(t); got != 1 {
		t.Fatalf("stored messages=%d, want only the saved student answer", got)
	}
	if !env.h.beginAnswer(env.threadID) {
		t.Fatal("cancellation left the thread locked")
	}
	env.h.endAnswer(env.threadID)
}

// TestSubmitUsesIndependentLLMContext не отменяет финальную оценку при закрытии браузера.
func TestSubmitUsesIndependentLLMContext(t *testing.T) {
	calls := 0
	env := newAnswerEnv(t, 3, contextEvaluator{grade: func(ctx context.Context) {
		calls++
		if ctx.Err() != nil || ctx.Done() != nil {
			t.Error("grading inherited browser cancellation")
		}
		if ctx.Value(requestMarker{}) != "submit" {
			t.Error("grading lost request values")
		}
	}})
	if _, err := env.store.AddMessage(model.Message{ThreadID: env.threadID, Role: model.RoleStudent, Content: "answer"}); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.WithValue(t.Context(), requestMarker{}, "submit"))
	cancel()
	req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/exam/%d/submit", env.sessID), nil).WithContext(ctx)
	rec := httptest.NewRecorder()
	env.router.ServeHTTP(rec, req)
	if rec.Code != http.StatusSeeOther {
		t.Fatalf("status=%d, body=%s", rec.Code, rec.Body.String())
	}
	if calls != 1 {
		t.Fatalf("grading calls=%d, want 1", calls)
	}
	sess, err := env.store.GetSession(env.sessID)
	if err != nil {
		t.Fatal(err)
	}
	if sess.Status != model.StatusGraded {
		t.Fatalf("status=%s, want graded", sess.Status)
	}
}
