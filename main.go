package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/fatih/color"
)

var (
	headerRegex = regexp.MustCompile(`\*1\|(?P<level>INFO|DEBUG|ERROR)\|0\|1\|[a-zA-Z0-9\-]+\|(?P<file>[a-zA-Z0-9\-_.:]+)\|(?P<time>[0-9/ .:]+)\|`)
	metricRegex = regexp.MustCompile(`_METRIC_`)

	infoColor      = color.New(color.FgGreen).SprintFunc()
	debugColor     = color.New(color.FgBlue).SprintFunc()
	errorColor     = color.New(color.FgRed).SprintFunc()
	highlightColor = color.New(color.BgYellow, color.FgRed).SprintFunc()
)

func applyOrHighlights(text string, highlights []string, baseColor func(a ...interface{}) string) string {
	if len(highlights) == 0 {
		return baseColor(text)
	}
	for _, word := range highlights {
		text = strings.Join(colorIt(strings.Split(text, word), baseColor), highlightColor(word))
	}
	return text
}

func applyAndHighlights(text string, highlights []string, baseColor func(a ...interface{}) string) string {
	found := len(highlights)
	if found == 0 {
		return baseColor(text)
	}
	if found == 1 {
		return applyOrHighlights(text, highlights, baseColor)
	}
	textNew := text
	for _, word := range highlights {
		if strings.Contains(text, word) {
			textNew = strings.Join(colorIt(strings.Split(textNew, word), baseColor), highlightColor(word))
			found--
		}
	}
	if found == 0 {
		return textNew
	}
	return baseColor(text)
}

func changeTimezone(text string) string {
	t, err := time.Parse("2006/01/02 15:04:05.000000", text)
	if err != nil {
		return "XX" + text + "XX"
	}
	return t.Local().Format("2006/01/02 15:04:05.000000")
}

func main() {
	highlightWords := flag.String("highlight", "", "Comma-separated list of words to highlight")
	highlightMode := flag.String("highlight-mode", "or", "Highlight mode: and, or")
	currentTimezone := flag.Bool("intoCurrentTimezone", false, "Change time to current timezone (input is taken in UTC)")
	flag.Parse()

	var highlights []string
	if *highlightWords != "" {
		highlights = strings.Split(*highlightWords, ",")
	}

	scanner := bufio.NewScanner(os.Stdin)
	const maxCapacity = 1024 * 1024 * 10
	buf := make([]byte, maxCapacity)
	scanner.Buffer(buf, maxCapacity)

	for scanner.Scan() {
		line := scanner.Text()

		if metricRegex.Match([]byte(line)) {
			continue
		}

		matches := headerRegex.FindStringSubmatch(line)
		if len(matches) < 3 {
			fmt.Println(line)
			continue
		}

		logLevel := matches[headerRegex.SubexpIndex("level")]
		time := matches[headerRegex.SubexpIndex("time")]
		file := matches[headerRegex.SubexpIndex("file")]
		line = headerRegex.ReplaceAllString(line, "")

		if *currentTimezone {
			time = changeTimezone(time)
		}

		line = fmt.Sprintf("%s\t%s\t%s\t%s", logLevel, time, file, line)

		var colorFunc = fmt.Sprint

		switch logLevel {
		case "INFO":
			colorFunc = infoColor
		case "DEBUG":
			colorFunc = debugColor
		case "ERROR":
			colorFunc = errorColor
		}

		switch *highlightMode {
		case "and":
			fmt.Println(applyAndHighlights(line, highlights, colorFunc))
		case "or":
			fmt.Println(applyOrHighlights(line, highlights, colorFunc))
		default:
			fmt.Println(applyOrHighlights(line, highlights, colorFunc))
		}
	}

	if err := scanner.Err(); err != nil {
		fmt.Fprintf(os.Stderr, "Error reading input: %v\n", err)
		os.Exit(1)
	}
}

func colorIt(ts []string, it func(a ...interface{}) string) []string {
	result := make([]string, len(ts))
	for i, t := range ts {
		result[i] = it(t)
	}
	return result
}
