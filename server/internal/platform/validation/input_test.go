package validation

import "testing"

func TestPlainTextAcceptsChineseEmojiAndRejectsMarkup(t *testing.T) {
	got, err := PlainText("  老大 🥂  ", TextOptions{Label: "昵称", MinRunes: 1, MaxRunes: 30})
	if err != nil || got != "老大 🥂" {
		t.Fatalf("unexpected valid result: %q %v", got, err)
	}
	if _, err := PlainText("online=1", TextOptions{Label: "备注", MaxRunes: 100}); err != nil {
		t.Fatalf("ordinary text was rejected: %v", err)
	}
	bad := []string{"<script>alert(1)</script>", `\"><img src=x onerror=alert(1)>`, "javascript:alert(1)", "&lt;script&gt;"}
	for _, value := range bad {
		if _, err := PlainText(value, TextOptions{Label: "昵称", MaxRunes: 100}); err == nil {
			t.Fatalf("dangerous value was accepted: %q", value)
		}
	}
}

func TestPhoneInviteAndURLValidation(t *testing.T) {
	if _, err := Phone("13812345678"); err != nil {
		t.Fatal(err)
	}
	if _, err := Phone("138123456789"); err == nil {
		t.Fatal("overlong phone was accepted")
	}
	if _, err := InviteCode("CODE1", false); err != nil {
		t.Fatal(err)
	}
	if _, err := HTTPURL("头像", "javascript:alert(1)", false); err == nil {
		t.Fatal("unsafe URL was accepted")
	}
	if _, err := OpaqueToken("登录凭证", `<script>alert(1)</script>`, 2048); err == nil {
		t.Fatal("unsafe token was accepted")
	}
}

func TestVerificationCodeRequiresSixDigits(t *testing.T) {
	got, err := VerificationCode(" 012345 ")
	if err != nil || got != "012345" {
		t.Fatalf("unexpected valid verification code: %q %v", got, err)
	}
	for _, value := range []string{"12345", "1234567", "12A456", "af445c74796e6c17"} {
		if _, err := VerificationCode(value); err == nil {
			t.Fatalf("invalid verification code was accepted: %q", value)
		}
	}
}

func TestSanitizeRichHTML(t *testing.T) {
	in := `<p class="intro" onclick="evil()">你好<script>alert(1)</script><img src="https://assets.example/a.jpg" onerror="evil()"><a href="javascript:evil()">链接</a></p>`
	want := `<p class="intro">你好<img src="https://assets.example/a.jpg" /><a>链接</a></p>`
	if got := SanitizeRichHTML(in); got != want {
		t.Fatalf("sanitize mismatch\nwant: %s\n got: %s", want, got)
	}
}
