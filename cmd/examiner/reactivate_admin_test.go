package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/pavelanni/examiner/internal/model"
	"github.com/pavelanni/examiner/internal/store"
)

func TestReactivateAdminRecovery(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "examiner.db")
	s, err := store.New(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	for _, user := range []model.User{
		{Username: "admin", PasswordHash: "original-hash", Role: model.UserRoleAdmin},
		{Username: "student", PasswordHash: "student-hash", Role: model.UserRoleStudent},
	} {
		if _, err := s.CreateUser(user); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := s.InsertQuestion(model.Question{Text: "Keep this question", MaxPoints: 10}); err != nil {
		t.Fatal(err)
	}
	if err := s.Close(); err != nil {
		t.Fatal(err)
	}
	for range 2 {
		var out bytes.Buffer
		cmd := rootCmd()
		cmd.SetOut(&out)
		cmd.SetErr(&out)
		cmd.SetArgs([]string{"reactivate-admin", "--db", dbPath})
		if err := cmd.Execute(); err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(out.String(), "Password unchanged") {
			t.Fatal("recovery should explain that the password was preserved")
		}
	}
	for _, username := range []string{"student", "missing"} {
		cmd := rootCmd()
		cmd.SetOut(&bytes.Buffer{})
		cmd.SetErr(&bytes.Buffer{})
		cmd.SetArgs([]string{"reactivate-admin", "--db", dbPath, "--username", username})
		if err := cmd.Execute(); err == nil {
			t.Fatalf("recovery should reject %q", username)
		}
	}
	s, err = store.New(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = s.Close() })
	admin, err := s.GetUserByUsername("admin")
	if err != nil || admin == nil || !admin.Active || admin.PasswordHash != "original-hash" || admin.Role != model.UserRoleAdmin {
		t.Fatal("recovery did not preserve the administrator", err)
	}
	student, err := s.GetUserByUsername("student")
	if err != nil || student == nil || student.Active || student.PasswordHash != "student-hash" || student.Role != model.UserRoleStudent {
		t.Fatal("recovery changed the non-admin user", err)
	}
	count, err := s.QuestionCount()
	if err != nil || count != 1 {
		t.Fatal("recovery changed exam questions", err)
	}
}

func TestReactivateAdminDoesNotCreateDatabase(t *testing.T) {
	path := filepath.Join(t.TempDir(), "missing.db")
	cmd := rootCmd()
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"reactivate-admin", "--db", path})
	if err := cmd.Execute(); err == nil {
		t.Fatal("expected an error for a missing database")
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("recovery created a database at the wrong path: %v", err)
	}
}
