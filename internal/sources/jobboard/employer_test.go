package jobboard

import (
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestEmployerNameFromTitle(t *testing.T) {
	t.Parallel()
	if name := EmployerNameFromTitle("Media Buyer в TapOK, віддалено | DOU"); name != "Media Buyer" {
		t.Fatalf("name=%q", name)
	}
	if name := EmployerNameFromTitle("DSP Media Buying Team Lead в TapOk – Djinni"); name != "DSP Media Buying Team Lead" {
		t.Fatalf("name=%q", name)
	}
}

func TestReverseDorks(t *testing.T) {
	t.Parallel()
	dorks := ReverseDorks("TapOK")
	if len(dorks) < 4 {
		t.Fatalf("dorks=%v", dorks)
	}
	hasTMe := false
	for _, dork := range dorks {
		if strings.Contains(dork, "site:t.me") {
			hasTMe = true
			break
		}
	}
	if !hasTMe {
		t.Fatalf("expected site:t.me dork for TapOK, got %v", dorks)
	}
}

func TestUpsertEmployerAndReverseDue(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), "employers.json")
	added, err := UpsertEmployer(path, EmployerEntry{Name: "TapOK", Slug: "tapok"})
	if err != nil || !added {
		t.Fatalf("added=%v err=%v", added, err)
	}
	f, err := LoadEmployers(path)
	if err != nil {
		t.Fatal(err)
	}
	due := EmployersDueForReverse(f, 7, 10)
	if len(due) != 1 || due[0].Name != "TapOK" {
		t.Fatalf("due=%v", due)
	}
	if err := MarkEmployerReversed(path, "tapok", time.Now().UTC()); err != nil {
		t.Fatal(err)
	}
	f2, _ := LoadEmployers(path)
	if len(EmployersDueForReverse(f2, 7, 10)) != 0 {
		t.Fatal("expected no due employers after reverse stamp")
	}
}

func TestSyncEmployersFromJobRegistry(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	jobPath := filepath.Join(dir, "jobs.json")
	empPath := filepath.Join(dir, "employers.json")
	if err := SaveRegistry(jobPath, File{URLs: []Entry{
		{
			URL:     "https://jobs.dou.ua/companies/tapok/vacancies/118950/",
			Title:   "Media Buyer в TapOK",
			Snippet: "keitaro gambling",
			Source:  "serp",
		},
	}}); err != nil {
		t.Fatal(err)
	}
	added, err := SyncEmployersFromJobRegistry(jobPath, empPath)
	if err != nil || added != 1 {
		t.Fatalf("added=%d err=%v", added, err)
	}
}
