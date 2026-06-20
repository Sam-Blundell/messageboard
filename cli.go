package main

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/Sam-Blundell/messageboard/board"
	"github.com/Sam-Blundell/messageboard/post"
)

type boardRepository interface {
	Create(name string) (board.Board, error)
	List() ([]board.Board, error)
}

// postRepository is the post-persistence port the cli depends on. It's declared
// here, at the consumer, so the cli needs only the behaviour it uses — the
// SQLite adapter (*post.SQLite) and test fakes both satisfy it, and neither the
// storage mechanism nor the on-disk shape leaks past this boundary. Threads and
// boards will get their own ports alongside this one.
type postRepository interface {
	Create(body string) (post.Post, error)
	ByID(id int64) (post.Post, error)
	List() ([]post.Post, error)
}

type cli struct {
	boards boardRepository
	posts  postRepository
	in     io.Reader
	out    io.Writer
	errOut io.Writer
}

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

func formatList(list []post.Post) (formattedList string) {
	if len(list) == 0 {
		return "no posts yet\n"
	}
	var formattedBuffer strings.Builder
	for _, p := range list {
		formattedBuffer.WriteString(formatPost(p))
	}
	formattedList = formattedBuffer.String()
	return formattedList
}

func (c *cli) handleGet(args string) (fetched post.Post, err error) {
	postID, err := strconv.ParseInt(args, 10, 64)
	if err != nil {
		return post.Post{}, fmt.Errorf("parsing argument: %w", err)
	}
	fetched, err = c.posts.ByID(postID)
	if err != nil {
		return post.Post{}, fmt.Errorf("can't get post %d: %w", postID, err)
	}
	return fetched, nil
}

func (c *cli) handlePost(body string) (newPost post.Post, err error) {
	if len(body) == 0 {
		return post.Post{}, errors.New("post requires a body")
	}
	newPost, err = c.posts.Create(body)
	if err != nil {
		return post.Post{}, err
	}
	return newPost, nil
}

func (c *cli) action(cmd, args string) (result string, quit bool, err error) {
	switch cmd {
	case "quit":
		return "", true, nil
	case "get":
		fetched, err := c.handleGet(args)
		if err != nil {
			return "", false, err
		}
		result = formatPost(fetched)
		return result, false, nil
	case "post":
		newPost, err := c.handlePost(args)
		if err != nil {
			return "", false, err
		}
		result = formatPost(newPost)
		return result, false, nil
	case "list":
		posts, err := c.posts.List()
		if err != nil {
			return "", false, err
		}
		result = formatList(posts)
		return result, false, nil
	case "":
		return "", false, nil
	default:
		err = fmt.Errorf("unknown command: %s", cmd)
		return "", false, err
	}
}

func (c *cli) run() {
	scanner := bufio.NewScanner(c.in)

	fmt.Fprint(c.out, ">")
	for scanner.Scan() {
		cmd, args := parseInput(scanner.Text())
		result, quit, err := c.action(cmd, args)
		if quit {
			return
		}
		if err != nil {
			fmt.Fprintln(c.errOut, err)
		}
		fmt.Fprint(c.out, result)
		fmt.Fprint(c.out, ">")
	}
	if err := scanner.Err(); err != nil {
		fmt.Fprintln(c.errOut, "reading input:", err)
	}
}
