package main

import (
	"bufio"
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/Sam-Blundell/messageboard/post"
)

func parseInput(input string) (cmd, args string) {
	trimmed := strings.TrimSpace(input)
	cmd, args, _ = strings.Cut(trimmed, " ")
	cmd = strings.ToLower(cmd)
	args = strings.TrimSpace(args)

	return cmd, args
}

func formatPost(p post.Post) (formattedPost string) {
	formattedTime := p.PostTime.Format("2006-01-02 15:04:05")
	formattedPost = fmt.Sprintf("%s - %d\n%s\n", formattedTime, p.ID, p.Body)
	return formattedPost
}

func handleGet(args string, postStore *post.Store) (formatted string, err error) {
	postID, err := strconv.ParseInt(args, 10, 64)
	if err != nil {
		return "", fmt.Errorf("parsing argument: %w", err)
	}
	fetched, err := postStore.ByID(postID)
	if err != nil {
		return "", fmt.Errorf("can't get post %d: %w", postID, err)
	}
	formatted = formatPost(fetched)
	return formatted, nil
}

func action(cmd, args string, postStore *post.Store) (result string, quit bool, err error) {
	switch cmd {
	case "quit":
		return "", true, nil
	case "get":
		result, err = handleGet(args, postStore)
		if err != nil {
			return "", false, err
		}
		return result, false, nil
	case "":
		return "", false, nil
	default:
		err = fmt.Errorf("unknown command: %s", cmd)
		return "", false, err
	}
}

func run(in io.Reader, out io.Writer, errOut io.Writer, postStorage *post.Store) {
	scanner := bufio.NewScanner(in)

	fmt.Fprint(out, ">")
	for scanner.Scan() {
		cmd, args := parseInput(scanner.Text())
		result, quit, err := action(cmd, args, postStorage)
		if quit {
			return
		}
		if err != nil {
			fmt.Fprintln(errOut, err)
		}
		fmt.Fprint(out, result)
		fmt.Fprint(out, ">")
	}
	if err := scanner.Err(); err != nil {
		fmt.Fprintln(errOut, "reading input:", err)
	}
}
