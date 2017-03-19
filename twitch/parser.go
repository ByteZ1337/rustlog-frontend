package twitch

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

type msgType int

const (
	PRIVMSG          msgType = iota + 1
	CLEARCHAT
	RANDOM
	EMOTE            = "EMOTE"
	BTTVEMOTE        = "BTTVEMOTE"
	BTTVCHANNELEMOTE = "BTTVCHANNELEMOTE"
)

type Message struct {
	Type        msgType
	Time        time.Time         `json:"time"`
	Channel     Channel           `json:"channel"`
	Username    string            `json:"username"`
	DisplayName string            `json:"displayName"`
	UserType    string            `json:"userType"`
	Color       string            `json:"color"`
	Badges      map[string]int    `json:"badges"`
	Emotes      []*Emote 		  `json:"emotes"`
	Tags        map[string]string `json:"tags"`
	Text        string            `json:"text"`
	Command		Command
}

type Emote struct {
	Name  string `json:"name"`
	ID    string `json:"id"`
	Type  string `json:"type"`
	Count int    `json:"count"`
}

func (bot *Bot) parseMessage(line string) *Message {
	if !strings.HasPrefix(line, "@") {
		return &Message{
			Text: line,
		}
	}
	spl := strings.SplitN(line, " :", 3)
	if len(spl) < 3 {
		return &Message{
			Text: line,
		}
	}
	tags, middle, text := spl[0], spl[1], spl[2]
	if strings.HasPrefix(text, "\u0001ACTION") {
		text = text[8 : len(text)-1]
	}
	msg := &Message{
		Time: time.Now(),
		Text: text,
		Tags: map[string]string{},
	}
	parseMiddle(msg, middle)
	parseTags(msg, tags[1:])
	if msg.Type == CLEARCHAT {
		msg.Username = "twitch"
		targetUser := msg.Text
		seconds, _ := strconv.Atoi(msg.Tags["ban-duration"])

		msg.Text = fmt.Sprintf("%s was timed out for %s: %s",
			targetUser,
			time.Duration(time.Duration(seconds)*time.Second),
			msg.Tags["ban-reason"])
	}
	msg.Command = parseCommand(msg.Text)
	msg = bot.addBttvEmotes(*msg)
	return msg
}

func parseMiddle(msg *Message, middle string) {
	for i, c := range middle {
		if c == '!' {
			msg.Username = middle[:i]
			middle = middle[i:]
		}
	}
	start := -1
	for i, c := range middle {
		if c == ' ' {
			if start == -1 {
				start = i + 1
			} else {
				typ := middle[start:i]
				switch typ {
				case "PRIVMSG":
					msg.Type = PRIVMSG
				case "CLEARCHAT":
					msg.Type = CLEARCHAT
				default:
					msg.Type = RANDOM
				}
				middle = middle[i:]
			}
		}
	}
	for i, c := range middle {
		if c == '#' {
			msg.Channel = NewChannel(middle[i+1:])
		}
	}
}

func parseTags(msg *Message, tagsRaw string) {
	tags := strings.Split(tagsRaw, ";")
	for _, tag := range tags {
		spl := strings.SplitN(tag, "=", 2)
		if len(spl) < 2 {
			return
		}
		value := strings.Replace(spl[1], "\\:", ";", -1)
		value = strings.Replace(value, "\\s", " ", -1)
		value = strings.Replace(value, "\\\\", "\\", -1)
		switch spl[0] {
		case "badges":
			msg.Badges = parseBadges(value)
		case "color":
			msg.Color = value
		case "display-name":
			msg.DisplayName = value
		case "emotes":
			msg.Emotes = parseTwitchEmotes(value, msg.Text)
		case "user-type":
			msg.UserType = value
		default:
			msg.Tags[spl[0]] = value
		}
	}
}

func parseBadges(badges string) map[string]int {
	m := map[string]int{}
	spl := strings.Split(badges, ",")
	for _, badge := range spl {
		s := strings.SplitN(badge, "/", 2)
		if len(s) < 2 {
			continue
		}
		n, _ := strconv.Atoi(s[1])
		m[s[0]] = n
	}
	return m
}

func parseTwitchEmotes(emoteTag, text string) []*Emote {
	emotes := []*Emote{}

	if emoteTag == "" {
		return emotes
	}

	runes := []rune(text)

	emoteSlice := strings.Split(emoteTag, "/")
	for i := range emoteSlice {
		spl := strings.Split(emoteSlice[i], ":")
		pos := strings.Split(spl[1], ",")
		sp := strings.Split(pos[0], "-")
		start, _ := strconv.Atoi(sp[0])
		end, _ := strconv.Atoi(sp[1])
		id := spl[0]
		e := &Emote{
			Type:  EMOTE,
			ID:    id,
			Count: strings.Count(emoteSlice[i], "-"),
			Name:  string(runes[start : end+1]),
		}

		emotes = append(emotes, e)
	}
	return emotes
}

func parseCommand(text string) Command {
	cmd := new(Command)

	if !strings.HasPrefix(text, "!") {
		cmd.IsCommand = false
		return *cmd
	}
	cmd.IsCommand = true

	argsFull := strings.Split(text, " ")
	cmd.Name = argsFull[0]
	args := append(argsFull[:0], argsFull[0+1:]...)
	cmd.Args = args

	return *cmd
}