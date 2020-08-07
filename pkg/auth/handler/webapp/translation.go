package webapp

import (
	"github.com/authgear/authgear-server/pkg/auth/config"
	"github.com/authgear/authgear-server/pkg/template"
)

const (
	TemplateItemTypeAuthUITranslationJSON config.TemplateItemType = "auth_ui_translation.json"
)

var TemplateAuthUITranslationJSON = template.Spec{
	Type: TemplateItemTypeAuthUITranslationJSON,
}