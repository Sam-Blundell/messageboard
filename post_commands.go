package main

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/Sam-Blundell/messageboard/post"
)

type postRepository interface {
	Create(body string) (post.Post, error)
	ByID(id int64) (post.Post, error)
	List() ([]post.Post, error)
}

func formatPost(p post.Post) (formattedPost string) {
	formattedTime := p.PostTime.Format("2006-01-02 15:04:05")
	formattedPost = fmt.Sprintf("%s - %d\n%s\n", formattedTime, p.ID, p.Body)
	return formattedPost
}

func formatPosts(list []post.Post) (formattedList string) {
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

type postCommands struct {
	posts postRepository
}

func (pc *postCommands) handleGet(tokens []string) (fetched post.Post, err error) {
	if len(tokens) != 1 {
		return post.Post{}, errors.New("post get expects an id number")
	}
	postID, err := strconv.ParseInt(tokens[0], 10, 64)
	if err != nil {
		return post.Post{}, fmt.Errorf("parsing argument: %w", err)
	}
	fetched, err = pc.posts.ByID(postID)
	if err != nil {
		return post.Post{}, fmt.Errorf("can't get post %d: %w", postID, err)
	}
	return fetched, nil
}

func (pc *postCommands) handleCreate(tokens []string) (newPost post.Post, err error) {
	if len(tokens) == 0 {
		return post.Post{}, errors.New("post requires a body")
	}
	body := strings.Join(tokens, " ")
	newPost, err = pc.posts.Create(body)
	if err != nil {
		return post.Post{}, err
	}
	return newPost, nil
}

func (pc *postCommands) handleList(tokens []string) (posts []post.Post, err error) {
	if len(tokens) != 0 {
		return posts, errors.New("post list takes no arguments")
	}
	return pc.posts.List()
}

func (pc *postCommands) dispatch(tokens []string) (result string, err error) {
	if len(tokens) == 0 {
		return "", ErrMissingCmd
	}

	action := tokens[0]

	switch strings.ToLower(action) {
	case "get":
		fetched, err := pc.handleGet(tokens[1:])
		if err != nil {
			return "", err
		}
		result = formatPost(fetched)
		return result, nil
	case "create":
		newPost, err := pc.handleCreate(tokens[1:])
		if err != nil {
			return "", err
		}
		result = formatPost(newPost)
		return result, nil
	case "list":
		posts, err := pc.handleList(tokens[1:])
		if err != nil {
			return "", err
		}
		result = formatPosts(posts)
		return result, nil
	default:
		return "", ErrUnknownCmd
	}
}
