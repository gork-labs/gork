package rules

import "testing"

func Test_splitInvocation_NoArgs(t *testing.T) {
	name, argstr, hasArgs, err := splitInvocation("admin")
	if err != nil || hasArgs || name != "admin" || argstr != "" {
		t.Fatalf("unexpected result: name=%q args=%q hasArgs=%v err=%v", name, argstr, hasArgs, err)
	}
}

func Test_splitInvocation_Unmatched(t *testing.T) {
	if _, _, _, err := splitInvocation("x("); err == nil {
		t.Fatal("expected unmatched '(' error")
	}
}
