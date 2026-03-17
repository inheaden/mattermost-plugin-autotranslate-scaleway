package main

import (
	"fmt"
	"strings"

	"github.com/pkg/errors"

	"github.com/mattermost/mattermost-server/v5/model"
	"github.com/mattermost/mattermost-server/v5/plugin"
)

const (
	translateUsername = "Translate Plugin"
	translateIconURL  = "https://docs.mattermost.com/_images/icon-76x76.png"
	enLanguage        = "en"
	autoLanguage      = "auto"
)

const commandHelp = `
This plugin is powered by Scaleway Generative APIs and a configured inference model to translate text on demand.

* |/translate on| - Add an option to translate a post with the default setting of Auto as source and English as target.
* |/translate off| - Remove an option to translate a post
* |/translate info| - Show user info on this plugin
* |/translate source [value]| - Update your translation source
  * |value| can be any of the plugin-supported language codes or "auto" to automatically detect language used.
* |/translate target [value]| - Update your translation target
  * |value| can be any of the plugin-supported language codes.
* |Language codes|: This plugin ships with a built-in language-code list for slash-command validation.
  `

// Below is plugin-maintained to keep slash-command validation and prompt generation stable.
var languageCodes = map[string]string{
	"auto":  "Auto",
	"af":    "Afrikaans",
	"sq":    "Albanian",
	"am":    "Amharic",
	"ar":    "Arabic",
	"hy":    "Armenian",
	"az":    "Azerbaijani",
	"bn":    "Bengali",
	"bs":    "Bosnian",
	"bg":    "Bulgarian",
	"ca":    "Catalan",
	"zh":    "Chinese (Simplified)",
	"zh-TW": "Chinese (Traditional)",
	"hr":    "Croatian",
	"cs":    "Czech",
	"da":    "Danish",
	"fa-AF": "Dari",
	"nl":    "Dutch",
	"en":    "English",
	"et":    "Estonian",
	"fa":    "Farsi (Persian)",
	"tl":    "Filipino Tagalog",
	"fi":    "Finnish",
	"fr":    "French",
	"fr-CA": "French (Canada)",
	"ka":    "Georgian",
	"de":    "German",
	"el":    "Greek",
	"gu":    "Gujarati",
	"ht":    "Haitian Creole",
	"ha":    "Hausa",
	"he":    "Hebrew",
	"hi":    "Hindi",
	"hu":    "Hungarian",
	"is":    "Icelandic",
	"id":    "Indonesian",
	"ga":    "Irish",
	"it":    "Italian",
	"ja":    "Japanese",
	"kn":    "Kannada",
	"kk":    "Kazakh",
	"ko":    "Korean",
	"lv":    "Latvian",
	"lt":    "Lithuanian",
	"mk":    "Macedonian",
	"ms":    "Malay",
	"ml":    "Malayalam",
	"mt":    "Maltese",
	"mn":    "Mongolian",
	"no":    "Norwegian (Bokmål)",
	"ps":    "Pashto",
	"pl":    "Polish",
	"pt":    "Portuguese (Brazil)",
	"pt-PT": "Portuguese (Portugal)",
	"pa":    "Punjabi",
	"ro":    "Romanian",
	"ru":    "Russian",
	"sr":    "Serbian",
	"si":    "Sinhala",
	"sk":    "Slovak",
	"sl":    "Slovenian",
	"so":    "Somali",
	"es":    "Spanish",
	"es-MX": "Spanish (Mexico)",
	"sw":    "Swahili",
	"sv":    "Swedish",
	"ta":    "Tamil",
	"te":    "Telugu",
	"th":    "Thai",
	"tr":    "Turkish",
	"uk":    "Ukrainian",
	"ur":    "Urdu",
	"uz":    "Uzbek",
	"vi":    "Vietnamese",
	"cy":    "Welsh",
}

func (p *Plugin) registerCommands() error {
	if err := p.API.RegisterCommand(&model.Command{
		Trigger:          "translate",
		DisplayName:      "Translate",
		Description:      "Mattermost Translate Plugin",
		AutoComplete:     true,
		AutoCompleteDesc: "Available commands: info, on, off, source, target, help",
		AutoCompleteHint: "[command]",
	}); err != nil {
		return errors.Wrap(err, "failed to register translate command")
	}

	return nil
}

func getCommandResponse(responseType, text string) *model.CommandResponse {
	return &model.CommandResponse{
		ResponseType: responseType,
		Text:         text,
		Username:     translateUsername,
		IconURL:      translateIconURL,
		Type:         model.POST_DEFAULT,
	}
}

func setUserInfoCommandResponse(userInfo *UserInfo, err *APIErrorResponse, action string) (*model.CommandResponse, *model.AppError) {
	var actionMapping = map[string]interface{}{
		"source": "setting up language source of the translate plugin",
		"target": "setting up language target of the translate plugin",
		"on":     "turning on the translate plugin",
		"off":    "turning off the translate plugin",
		"info":   "getting user information",
	}

	if err != nil {
		errorMessage := ""
		if len(err.Message) > 0 {
			errorMessage = err.Message
		}

		text := fmt.Sprintf("An error occurred %s. `%s`", actionMapping[action], errorMessage)
		if err.ID == apiErrorNoRecordFound {
			text = "No record found. If not yet turned on for the first time, try `/translate on` to enable."
		}

		return getCommandResponse(model.COMMAND_RESPONSE_TYPE_EPHEMERAL, text), nil
	}

	text := fmt.Sprintf(
		"Successfully updated!\nYour translate plugin settings:\n * Active: `%s`\n * Language: `source: %s`, `target: %s`\n",
		userInfo.getActivatedString(), languageCodes[userInfo.SourceLanguage], languageCodes[userInfo.TargetLanguage],
	)

	if action == "off" {
		text = "Translate plugin is turned off."
	}

	return getCommandResponse(model.COMMAND_RESPONSE_TYPE_EPHEMERAL, text), nil
}

// ExecuteCommand executes a command that has been previously registered via the RegisterCommand API.
func (p *Plugin) ExecuteCommand(c *plugin.Context, args *model.CommandArgs) (*model.CommandResponse, *model.AppError) {
	split := strings.Fields(args.Command)
	command := split[0]
	action := ""
	param := ""
	if len(split) > 1 {
		action = split[1]
	}
	if len(split) > 2 {
		param = split[2]
	}

	if command != "/translate" {
		return nil, nil
	}

	var text = ""
	if action == "" || action == "help" {
		text = "###### Mattermost Translate Plugin - Slash Command Help\n" + strings.Replace(commandHelp, "|", "`", -1)
		return getCommandResponse(model.COMMAND_RESPONSE_TYPE_EPHEMERAL, text), nil
	}

	userInfo, err := p.getUserInfo(args.UserId)
	if userInfo == nil && action != "on" {
		text = "No record found. Try `/translate on` to enable."
		return getCommandResponse(model.COMMAND_RESPONSE_TYPE_EPHEMERAL, text), nil
	}

	switch action {
	case "info":
		text = fmt.Sprintf(
			"Your translate plugin settings:\n * Active: `%s`\n * Language: `source: %s`, `target: %s`\n",
			userInfo.getActivatedString(), languageCodes[userInfo.SourceLanguage], languageCodes[userInfo.TargetLanguage],
		)
		return getCommandResponse(model.COMMAND_RESPONSE_TYPE_EPHEMERAL, text), nil
	case "on":
		if userInfo == nil {
			userInfo = p.NewUserInfo(args.UserId)
		} else {
			userInfo.Activated = true
		}

		err = p.setUserInfo(userInfo)
		return setUserInfoCommandResponse(userInfo, err, action)
	case "off":
		if userInfo == nil {
			return getCommandResponse(model.COMMAND_RESPONSE_TYPE_EPHEMERAL, "No record found. If not yet turned on for the first time, try `/translate on` to enable. Otherwise, your record is lost for unknown reason."), nil
		}

		userInfo.Activated = false
		err = p.setUserInfo(userInfo)
		return setUserInfoCommandResponse(userInfo, err, action)
	case "source":
		if userInfo == nil {
			return getCommandResponse(model.COMMAND_RESPONSE_TYPE_EPHEMERAL, "No record found. If not yet turned on for the first time, try `/translate on` to enable. Otherwise, your record is lost for unknown reason."), nil
		}

		if param == "" {
			return getCommandResponse(model.COMMAND_RESPONSE_TYPE_EPHEMERAL, "Invalid empty source language. Should pass a valid language code or set to \"auto\"."), nil
		}

		if languageCodes[param] == "" {
			return getCommandResponse(model.COMMAND_RESPONSE_TYPE_EPHEMERAL, fmt.Sprintf("Invalid \"%s\" source language. Should pass a valid language code or set to \"auto\".", param)), nil
		}

		userInfo.SourceLanguage = param
		err = p.setUserInfo(userInfo)
		return setUserInfoCommandResponse(userInfo, err, action)
	case "target":
		if userInfo == nil {
			return getCommandResponse(model.COMMAND_RESPONSE_TYPE_EPHEMERAL, "No record found. If not yet turned on for the first time, try `/translate on` to enable."), nil
		}

		if param == "" {
			return getCommandResponse(model.COMMAND_RESPONSE_TYPE_EPHEMERAL, "Invalid empty target language. Should pass a valid language code."), nil
		}

		if param == "auto" {
			return getCommandResponse(model.COMMAND_RESPONSE_TYPE_EPHEMERAL, "Target language can't be set to \"auto\". Should pass a valid language code."), nil
		}

		if languageCodes[param] == "" {
			return getCommandResponse(model.COMMAND_RESPONSE_TYPE_EPHEMERAL, fmt.Sprintf("Invalid \"%s\" target language. Should pass a valid language code.", param)), nil
		}

		userInfo.TargetLanguage = param
		err = p.setUserInfo(userInfo)
		return setUserInfoCommandResponse(userInfo, err, action)
	default:
		text = "###### Mattermost Translate Plugin - Slash Command Help\n" + strings.Replace(commandHelp, "|", "`", -1)
		return getCommandResponse(model.COMMAND_RESPONSE_TYPE_EPHEMERAL, text), nil
	}
}
