package qcode

import "testing"

func TestDecodeRecognized(t *testing.T) {
	d := Decode("QMRLC")
	if !d.Recognized {
		t.Fatal("QMRLC باید شناسایی شود")
	}
	if d.Subject != "MR" || d.Condition != "LC" {
		t.Errorf("تفکیک نادرست: subject=%q condition=%q", d.Subject, d.Condition)
	}
	if d.Category != CatRunway {
		t.Errorf("category=%q، انتظار RUNWAY", d.Category)
	}
	if d.ConditionLabel != "Closed" {
		t.Errorf("condition label=%q، انتظار Closed", d.ConditionLabel)
	}
}

func TestDecodeFallbackByFirstLetter(t *testing.T) {
	// QM?? با موضوع ناشناختهٔ MZ → fallback به حرف M (movement area → RUNWAY)
	d := Decode("QMZLC")
	if d.Recognized {
		t.Error("MZ نباید کاملاً شناسایی شود")
	}
	if d.Category != CatRunway {
		t.Errorf("fallback category=%q، انتظار RUNWAY", d.Category)
	}
}

func TestDecodeInvalid(t *testing.T) {
	d := Decode("XYZ")
	if d.Recognized || d.Category != CatOther {
		t.Errorf("کد نامعتبر باید OTHER و ناشناخته باشد: %+v", d)
	}
}

func TestExtractPriority(t *testing.T) {
	// فیلد qcode اولویت دارد
	if got := Extract("QNVAS", "Q) X/QMRLC/...", ""); got != "QNVAS" {
		t.Errorf("اولویت اشتباه: %q", got)
	}
}
