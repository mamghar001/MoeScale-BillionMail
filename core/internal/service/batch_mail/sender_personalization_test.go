package batch_mail

import (
	"billionmail-core/internal/model/entity"
	"context"
	"strings"
	"testing"
)

func TestPersonalizationRotatedSender(t *testing.T) {
	engine := GetTemplateEngine()
	ctx := context.Background()

	task := &entity.EmailTask{
		Id:        999,
		Addresser: "isabelle.reynolds@aisalesbox.shop",
		FullName:  "Isabelle Reynolds",
		Subject:   "Quick question for {{ .Subscriber.Email }}",
	}

	contact := &entity.Contact{
		Email: "lead@example.com",
	}

	// Test 1: {{ .Task.FullName }}
	templateContent1 := "<p>Hi there,</p><p>Best regards,<br>{{ .Task.FullName }}</p>"
	rendered1, err := engine.RenderEmailTemplate(ctx, templateContent1, contact, task, "")
	if err != nil {
		t.Fatalf("RenderEmailTemplate failed: %v", err)
	}
	if !strings.Contains(rendered1, "Isabelle Reynolds") {
		t.Errorf("Expected 'Isabelle Reynolds' in rendered content, got: %s", rendered1)
	}

	// Test 2: {{ .Task.FirstName }} and {{ .Task.LastName }}
	templateContent2 := "<p>Cheers,<br>{{ .Task.FirstName }} {{ .Task.LastName }}</p>"
	rendered2, err := engine.RenderEmailTemplate(ctx, templateContent2, contact, task, "")
	if err != nil {
		t.Fatalf("RenderEmailTemplate failed: %v", err)
	}
	if !strings.Contains(rendered2, "Isabelle Reynolds") {
		t.Errorf("Expected 'Isabelle Reynolds' from First + Last name, got: %s", rendered2)
	}

	// Test 3: {{ .Sender.FullName }} and {{ .Sender.Email }}
	templateContent3 := "<p>Signed,<br>{{ .Sender.FullName }} ({{ .Sender.Email }})</p>"
	rendered3, err := engine.RenderEmailTemplate(ctx, templateContent3, contact, task, "")
	if err != nil {
		t.Fatalf("RenderEmailTemplate failed: %v", err)
	}
	if !strings.Contains(rendered3, "Isabelle Reynolds (isabelle.reynolds@aisalesbox.shop)") {
		t.Errorf("Expected 'Isabelle Reynolds (isabelle.reynolds@aisalesbox.shop)', got: %s", rendered3)
	}

	// Test 4: Subject line personalization
	renderedSubj, err := engine.RenderEmailTemplate(ctx, task.Subject, contact, task, "")
	if err != nil {
		t.Fatalf("RenderEmailTemplate failed: %v", err)
	}
	if !strings.Contains(renderedSubj, "lead@example.com") {
		t.Errorf("Expected 'lead@example.com' in subject, got: %s", renderedSubj)
	}

	// Test 5: Spintax + personalization
	templateContent5 := "{Hi|Hello|Hey} {{ .Subscriber.FirstName }}, this is {{ .Task.FullName }} from {{ .Subscriber.Company }}."
	contact5 := &entity.Contact{
		Email: "john@example.com",
		Attribs: map[string]string{
			"first_name": "John",
			"company":    "Acme Corp",
		},
	}
	rendered5, err := engine.RenderEmailTemplate(ctx, templateContent5, contact5, task, "")
	if err != nil {
		t.Fatalf("RenderEmailTemplate failed: %v", err)
	}
	if !strings.Contains(rendered5, "John") {
		t.Errorf("Expected 'John' in rendered content, got: %s", rendered5)
	}
	if !strings.Contains(rendered5, "Isabelle Reynolds") {
		t.Errorf("Expected 'Isabelle Reynolds' in rendered content, got: %s", rendered5)
	}
	if !strings.Contains(rendered5, "Acme Corp") {
		t.Errorf("Expected 'Acme Corp' in rendered content, got: %s", rendered5)
	}
	if strings.Contains(rendered5, "{") || strings.Contains(rendered5, "|") {
		t.Errorf("Expected spintax to be parsed completely, got: %s", rendered5)
	}
}

