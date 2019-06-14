package template

import (
	"testing"

	"github.com/franela/goreq"

	"github.com/h2non/gock"
	"github.com/skygeario/skygear-server/pkg/core/config"
	"github.com/skygeario/skygear-server/pkg/core/template"
	. "github.com/smartystreets/goconvey/convey"
)

func TestResetPasswordPayload(t *testing.T) {
	Convey("Test auth template", t, func() {
		config := config.TenantConfiguration{
			WelcomeEmail: config.WelcomeEmailConfiguration{
				HTMLURL: "http://template.com/welcome-email-html-url",
			},
			UserVerify: config.UserVerifyConfiguration{
				KeyConfigs: []config.UserVerifyKeyConfiguration{
					config.UserVerifyKeyConfiguration{
						Key: "key1",
						ProviderConfig: config.UserVerifyKeyProviderConfiguration{
							TextURL: "http://template.com/userverify-key1-text-url",
						},
					},
				},
			},
		}

		templateEngine := template.NewEngine()
		RegisterDefaultTemplates(templateEngine)
		templateEngine = NewEngineWithConfig(templateEngine, config)

		context := map[string]interface{}{
			"user": map[string]map[string]interface{}{
				"LoginIDs": {
					"username": "chima",
				},
			},
		}

		gock.InterceptClient(goreq.DefaultClient)
		defer gock.Off()
		defer gock.RestoreClient(goreq.DefaultClient)

		Convey("render default template", func() {
			out, err := templateEngine.ParseTextTemplate(TemplateNameWelcomeEmailText, context, template.ParseOption{})
			So(err, ShouldBeNil)
			So(out, ShouldEqual, `Hello chima,

Welcome to Skygear.

Thanks.`)
		})

		Convey("render template specified in template", func() {
			gock.New("http://template.com").
				Get("/welcome-email-html-url").
				Reply(200).
				BodyString("content of welcome-email-html-url")

			out, err := templateEngine.ParseTextTemplate(TemplateNameWelcomeEmailHTML, context, template.ParseOption{})
			So(err, ShouldBeNil)
			So(out, ShouldEqual, "content of welcome-email-html-url")
			So(gock.IsDone(), ShouldBeTrue)
		})

		Convey("render template of specific verify key", func() {
			gock.New("http://template.com").
				Get("/userverify-key1-text-url").
				Reply(200).
				BodyString("content of userverify-key1-text-url")

			out, err := templateEngine.ParseTextTemplate(VerifyTextTemplateNameForKey("key1"), context, template.ParseOption{})
			So(err, ShouldBeNil)
			So(out, ShouldEqual, "content of userverify-key1-text-url")
			So(gock.IsDone(), ShouldBeTrue)
		})
	})
}