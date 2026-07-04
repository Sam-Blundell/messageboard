package main

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/Sam-Blundell/messageboard/post"
)

// postCreator is the port for creating posts — satisfied by core.Core, whose
// CreatePost also bumps the owning thread atomically.
type postCreator interface {
	CreatePost(threadID int64, body string) (post.Post, error)
}

type postRepository interface {
	ByID(id int64) (post.Post, error)
	List() ([]post.Post, error)
}

func formatPost(p post.Post) string {
	formattedTime := p.PostTime.Format("2006-01-02 15:04:05")
	formatted := fmt.Sprintf("%s - %d\n%s\n", formattedTime, p.ID, p.Body)
	return formatted
}

func formatPosts(list []post.Post) string {
	if len(list) == 0 {
		return "no posts yet\n"
	}
	var formattedBuffer strings.Builder
	for _, p := range list {
		formattedBuffer.WriteString(formatPost(p))
	}
	return formattedBuffer.String()
}

type postCommands struct {
	creator postCreator
	posts   postRepository
}

func (pc *postCommands) handleGet(tokens []string) (post.Post, error) {
	if len(tokens) != 1 {
		return post.Post{}, errors.New("usage: post get <post-id>")
	}
	postID, err := strconv.ParseInt(tokens[0], 10, 64)
	if err != nil {
		return post.Post{}, fmt.Errorf("post ID must be a number, got %q", tokens[0])
	}
	fetched, err := pc.posts.ByID(postID)
	if err != nil {
		return post.Post{}, fmt.Errorf("can't get post %d: %w", postID, err)
	}
	return fetched, nil
}

func (pc *postCommands) handleCreate(tokens []string) (post.Post, error) {
	if len(tokens) != 2 {
		return post.Post{}, errors.New("usage: post create <thread-id> <body> (quote a body containing spaces)")
	}
	threadID, err := strconv.ParseInt(tokens[0], 10, 64)
	if err != nil {
		return post.Post{}, fmt.Errorf("thread ID must be a number, got %q", tokens[0])
	}
	body := tokens[1]
	newPost, err := pc.creator.CreatePost(threadID, body)
	return newPost, err
}

func (pc *postCommands) handleList(tokens []string) ([]post.Post, error) {
	if len(tokens) != 0 {
		return nil, errors.New("post list takes no arguments")
	}
	posts, err := pc.posts.List()
	return posts, err
}

func (pc *postCommands) dispatch(tokens []string) (string, error) {
	if len(tokens) == 0 {
		return "", ErrMissingCmd
	}

	action := strings.ToLower(tokens[0])

	switch action {
	case "get":
		fetched, err := pc.handleGet(tokens[1:])
		if err != nil {
			return "", err
		}
		return formatPost(fetched), nil
	case "create":
		newPost, err := pc.handleCreate(tokens[1:])
		if err != nil {
			return "", err
		}
		return formatPost(newPost), nil
	case "list":
		posts, err := pc.handleList(tokens[1:])
		if err != nil {
			return "", err
		}
		return formatPosts(posts), nil
	default:
		return "", ErrUnknownCmd
	}
}
