package validate_test

import (
	"fmt"
	"testing"

	"github.com/workoak/wop/wou/validate"
)

func TestNewBucket(t *testing.T) {
	bucket := validate.NewBucket()
	if bucket == nil {
		t.Fatal("expoecting validator")
	}
	_, err := bucket.Validate("valid blank", "", validate.Blank)
	if err != nil {
		t.Fatal(err)
	}

	if !bucket.IsValid() {
		t.Fatalf("wanted valid, got not valid")
	}
	bucket.Validate("invalid blank", "not blank", validate.Blank)
	bucket.Validate("invalid not blank", "", validate.NotBlank)
	if bucket.IsValid() {
		t.Fatalf("wanted invalid, got valid")
	}
	if bucket.ErrorCount() != 2 {
		t.Fatalf("wanted 2 errors, got %v", bucket.ErrorCount())
	}
	msg := fmt.Sprintf("errors:%v", bucket.Result().ValidationErrors())
	fmt.Println(msg)

}

func TestOneOfRule(t *testing.T) {
	bucket := validate.NewBucket()
	bucket.Validate("valid oneof", "go", validate.OneOfRule("go", "js"))
	if !bucket.IsValid() {
		t.Fatalf("wanted valid, got not valid")
	}
	bucket.Validate("invalid oneof", "java", validate.OneOfRule("go", "js"))
	if bucket.IsValid() {
		t.Fatalf("wanted invalid, got valid")
	}
	if bucket.ErrorCount() != 1 {
		t.Fatalf("wanted 1 errors, got %v", bucket.ErrorCount())
	}
}

func TestRegExRule(t *testing.T) {
	bucket := validate.NewBucket()
	bucket.Validate("valid regex", "go", validate.RegExRule("go|js"))
	if !bucket.IsValid() {
		t.Fatalf("wanted valid, got not valid")
	}
	bucket.Validate("invalid regex", "java", validate.RegExRule("go"))
	if bucket.IsValid() {
		t.Fatalf("wanted invalid, got valid")
	}
	if bucket.ErrorCount() != 1 {
		t.Fatalf("wanted 1 errors, got %v", bucket.ErrorCount())
	}
}
