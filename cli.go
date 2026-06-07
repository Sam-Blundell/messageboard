package main

import (
	"bufio"
	"fmt"
	"io"
	"strconv"
	"strings"
)

func parseInput(input string) (cmd, args string) {
	trimmed := strings.TrimSpace(input)
	cmd, args, _ = strings.Cut(trimmed, " ")
	cmd = strings.ToLower(cmd)
	args = strings.TrimSpace(args)

	return cmd, args
}

func formatPost(post Post) (formattedPost string) {
	formattedTime := post.PostTime.Format("2006-01-02 15:04:05")
	formattedPost = fmt.Sprintf("%s - %d\n%s\n", formattedTime, post.ID, post.Body)
	return formattedPost
}

func handleGet(args string, postStore *PostStorage) (formatted string, err error) {
	postID, err := strconv.ParseInt(args, 10, 64)
	if err != nil {
		return "", fmt.Errorf("parsing argument: %w", err)
	}
	post, err := postStore.ByID(postID)
	if err != nil {
		return "", fmt.Errorf("can't get post %d: %w", postID, err)
	}
	formatted = formatPost(post)
	return formatted, nil
}

func action(cmd, args string, postStore *PostStorage) (result string, quit bool, err error) {
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

func run(in io.Reader, out io.Writer, errOut io.Writer, postStorage *PostStorage) {
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
