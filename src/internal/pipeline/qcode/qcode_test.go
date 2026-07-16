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
	// موضوع ناشناختهٔ MZ → fallback به گروه M.
	// باید به دستهٔ خنثای MOVEMENT_AREA برود، نه RUNWAY: نگاشتِ حدسی به پرخطرترین
	// دستهٔ گروه، اهمیت را به‌غلط تا سطح بحرانی تشدید می‌کند.
	d := Decode("QMZLC")
	if d.Recognized {
		t.Error("MZ نباید کاملاً شناسایی شود")
	}
	if d.Category == CatRunway {
		t.Error("fallback نباید به RUNWAY تشدید شود")
	}
	if d.Category != CatMovementArea {
		t.Errorf("fallback category=%q، انتظار MOVEMENT_AREA", d.Category)
	}
}

func TestDecodeRapidExitTaxiway(t *testing.T) {
	d := Decode("QMYLC")
	if !d.Recognized || d.Category != CatTaxiway {
		t.Errorf("QMYLC باید تاکسی‌وی شناسایی شود: %+v", d)
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
