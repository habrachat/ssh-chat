package message

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/kenshaw/emoji"
)

// Message is an interface for messages.
type Message interface {
	Render(*Theme) string
	String() string
	Command() string
	Timestamp() time.Time
}

type MessageTo interface {
	Message
	To() *User
}

type MessageFrom interface {
	Message
	From() *User
	OriginalFrom() *User
}

func ParseInput(body string, from *User) Message {
	m := NewPublicMsg(body, from, from)
	cmd, isCmd := m.ParseCommand()
	if isCmd {
		return cmd
	}
	return m
}

// Msg is a base type for other message types.
type Msg struct {
	body      string
	timestamp time.Time
	// TODO: themeCache *map[*Theme]string
}

func NewMsg(body string) *Msg {
	return &Msg{
		body:      body,
		timestamp: time.Now(),
	}
}

// Render message based on a theme.
func (m Msg) Render(t *Theme) string {
	// TODO: Render based on theme
	// TODO: Cache based on theme
	return m.String()
}

func (m Msg) String() string {
	return m.body
}

func (m Msg) Command() string {
	return ""
}

func (m Msg) Timestamp() time.Time {
	return m.timestamp
}

// PublicMsg is any message from a user sent to the room.
type PublicMsg struct {
	Msg
	from *User
	originalFrom *User
}

func NewPublicMsg(body string, from *User, originalFrom *User) PublicMsg {
	return PublicMsg{
		Msg: Msg{
			body:      body,
			timestamp: time.Now(),
		},
		from: from,
		originalFrom: originalFrom,
	}
}

func (m PublicMsg) From() *User {
	return m.from
}

func (m PublicMsg) OriginalFrom() *User {
	return m.originalFrom
}

func (m PublicMsg) ParseCommand() (*CommandMsg, bool) {
	// Check if the message is a command
	if !strings.HasPrefix(m.body, "/") {
		return nil, false
	}

	// Parse
	// TODO: Handle quoted fields properly
	fields := strings.Fields(m.body)
	command, args := fields[0], fields[1:]
	msg := CommandMsg{
		PublicMsg: m,
		command:   command,
		args:      args,
	}
	return &msg, true
}


func renderMarkdown(s string) string {
	s = emoji.ReplaceAliases(s)

	re := regexp.MustCompile(`\[([^\]]+)\]\(([^) ]+)\)`)
	s = re.ReplaceAllString(s, "\x01\x1b]8;;$2\x1b\\\x02$1\x01\x1b]8;;\x1b\\\x02")

	result := ""
	is_bold := false
	is_italic := false
	inside_backslash := false
	inside_escape := false
	for i, c := range s {
		if inside_escape {
			if c == '\x02' {
				inside_escape = false
			} else {
				result += string(c)
			}
			continue
		}
		if c == '\x01' {
			inside_escape = true
			continue
		}
		if inside_backslash {
			inside_backslash = false
			result += string(c)
			continue
		}
		if c == '\\' {
			inside_backslash = true
		} else if c == '*' && !is_bold && (i == 0 || s[i - 1] == ' ') && (i + 1 < len(s) && s[i + 1] != ' ') {
			result += "\x1b[1m"
			is_bold = true
		} else if c == '*' && is_bold && (i > 0 && s[i - 1] != ' ') && (i + 1 == len(s) || s[i + 1] == ' ') {
			result += "\x1b[22m"
			is_bold = false
		} else if c == '_' && !is_italic && (i == 0 || s[i - 1] == ' ') && (i + 1 < len(s) && s[i + 1] != ' ') {
			result += "\x1b[3m"
			is_italic = true
		} else if c == '_' && is_italic && (i > 0 && s[i - 1] != ' ') && (i + 1 == len(s) || s[i + 1] == ' ') {
			result += "\x1b[23m"
			is_italic = false
		} else {
			result += string(c)
		}
	}
	if inside_backslash {
		result += "\\"
	}
	result += "\x1b[0m"
	return result
}


func renderMessageFor(prefix string, u *User, sep string, body string, t *Theme, cfg *UserConfig, doHighlight bool) string {
	if cfg != nil && !cfg.ApiMode {
		body = renderMarkdown(body)
	}
	if t != nil && cfg != nil && doHighlight {
		newBody := cfg.Highlight.ReplaceAllString(body, t.Highlight("${1}"))
		if newBody != body {
			body = newBody
			if cfg.Bell {
				body += Bel
			}
		}
	}
	if t == nil {
		return prefix + u.Name() + sep + body
	} else {
		return prefix + t.ColorName(u) + sep + body
	}
}


func (m PublicMsg) Render(t *Theme) string {
	return renderMessageFor("", m.from, ": ", m.body, t, nil, true)
}

// RenderFor renders the message for other users to see.
func (m PublicMsg) RenderFor(cfg UserConfig) string {
	return renderMessageFor("", m.from, ": ", m.body, cfg.Theme, &cfg, true)
}

// RenderSelf renders the message for when it's echoing your own message.
func (m PublicMsg) RenderSelf(cfg UserConfig) string {
	return renderMessageFor("[", m.from, "] ", m.body, cfg.Theme, &cfg, false)
}

func (m PublicMsg) String() string {
	return fmt.Sprintf("%s: %s", m.from.Name(), m.body)
}

// EmoteMsg is a /me message sent to the room.
type EmoteMsg struct {
	Msg
	from *User
	originalFrom *User
}

func NewEmoteMsg(body string, from *User, originalFrom *User) *EmoteMsg {
	return &EmoteMsg{
		Msg: Msg{
			body:      body,
			timestamp: time.Now(),
		},
		from: from,
		originalFrom: originalFrom,
	}
}

func (m EmoteMsg) From() *User {
	return m.from
}

func (m EmoteMsg) OriginalFrom() *User {
	return m.originalFrom
}

func (m EmoteMsg) Render(t *Theme) string {
	return renderMessageFor("** ", m.from, " ", m.body, t, nil, true)
}

func (m EmoteMsg) String() string {
	return m.Render(nil)
}

// PrivateMsg is a message sent to another user, not shown to anyone else.
type PrivateMsg struct {
	PublicMsg
	to *User
}

func NewPrivateMsg(body string, from *User, to *User) PrivateMsg {
	return PrivateMsg{
		PublicMsg: NewPublicMsg(body, from, from),
		to:        to,
	}
}

func (m PrivateMsg) To() *User {
	return m.to
}

func (m PrivateMsg) From() *User {
	return m.from
}

func (m PrivateMsg) OriginalFrom() *User {
	return m.originalFrom
}

func (m PrivateMsg) Render(t *Theme) string {
	return renderMessageFor("[PM from ", m.from, "] ", m.body, t, nil, true)
}

func (m PrivateMsg) String() string {
	return m.Render(nil)
}

// SystemMsg is a response sent from the server directly to a user, not shown
// to anyone else. Usually in response to something, like /help.
type SystemMsg struct {
	Msg
	to *User
}

func NewSystemMsg(body string, to *User) *SystemMsg {
	return &SystemMsg{
		Msg: Msg{
			body:      body,
			timestamp: time.Now(),
		},
		to: to,
	}
}

func (m *SystemMsg) Render(t *Theme) string {
	if t == nil {
		return m.String()
	}
	return t.ColorSys(m.String())
}

func (m *SystemMsg) String() string {
	return fmt.Sprintf("-> %s", m.body)
}

func (m *SystemMsg) To() *User {
	return m.to
}

// AnnounceMsg is a message sent from the server to everyone, like a join or
// leave event.
type AnnounceMsg struct {
	Msg
}

func NewAnnounceMsg(body string) *AnnounceMsg {
	return &AnnounceMsg{
		Msg: Msg{
			body:      body,
			timestamp: time.Now(),
		},
	}
}

func (m AnnounceMsg) Render(t *Theme) string {
	if t == nil {
		return m.String()
	}
	return t.ColorSys(m.String())
}

func (m AnnounceMsg) String() string {
	return fmt.Sprintf(" * %s", m.body)
}

// MOTDMsg is a MOTD message
type MOTDMsg struct {
	Msg
}

func NewMOTDMsg(body string) *MOTDMsg {
	return &MOTDMsg{
		Msg: Msg{
			body:      body,
			timestamp: time.Now(),
		},
	}
}

func (m MOTDMsg) Render(t *Theme) string {
	if t == nil {
		return m.String()
	}
	return t.ColorSys(m.String())
}

func (m MOTDMsg) String() string {
	return fmt.Sprintf(" %s", m.body)
}

type CommandMsg struct {
	PublicMsg
	command string
	args    []string
}

func (m CommandMsg) Command() string {
	return m.command
}

func (m CommandMsg) Args() []string {
	return m.args
}

func (m CommandMsg) Body() string {
	return m.body
}
