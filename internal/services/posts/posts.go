package posts

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"ozon/internal/domain/models"
	"ozon/internal/storage"
	"ozon/pkg/logger/sl"
)

// Post - структура для работы с постом
type Post struct {
	Log          *slog.Logger
	PostSaver    PostSaver
	PostProvider PostProvider
}

// PostSaver - интерфейс для сохранения поста в базе данных
type PostSaver interface {
	SavePost(ctx context.Context, userID int64, title, content string, allowComments bool) (int64, time.Time, error)
}

// PostProvider - интерфейс для получения поста
type PostProvider interface {
	ProvidePost(ctx context.Context, postID int64) (models.Post, error)
	ProvideAllPosts(ctx context.Context, page int64) ([]models.Post, error)
}

var (
	ErrPostExists     = errors.New("post with the same title already exists")
	ErrPostNotFound   = errors.New("post does not exist")
	ErrGetPosts       = errors.New("failed to get posts")
	ErrConnectionTime = errors.New("connection time expired")
	ErrGetComments    = errors.New("failed to get comments")
	ErrFoundComments  = errors.New("no comments found")
)

// New возвращает структуру для работы с постом
func New(log *slog.Logger, postSaver PostSaver, postProvider PostProvider) *Post {
	return &Post{
		Log:          log,
		PostSaver:    postSaver,
		PostProvider: postProvider,
	}
}

// PostNew публикует новый пост от имени пользователя
// Возвращает ошибку, если статья с таким названием уже существует,
// либо не удалось сделать запись в БД или если превышено время ожмдания ответа от БД
func (p *Post) PostNew(ctx context.Context, userID int64, title, content string, allowComments bool) (int64, time.Time, error) {
	const op = "internal/services/posts.PostNew"

	log := p.Log.With(slog.String("op", op), slog.Int64("user_id", userID))
	log.Info("attempting to create new post")

	postID, createdAt, err := p.PostSaver.SavePost(ctx, userID, title, content, allowComments)
	if err != nil {
		if errors.Is(err, storage.ErrPostExists) {
			log.Warn("post already exists", sl.Err(err))
			return 0, time.Time{}, fmt.Errorf("%s: %w", op, ErrPostExists)
		}

		if errors.Is(err, storage.ErrConnectionTime) {
			log.Error("connection time expired", sl.Err(err))
			return 0, time.Time{}, fmt.Errorf("%s: %w", op, ErrConnectionTime)
		}

		log.Error("failed to save post", sl.Err(err))
		return 0, time.Time{}, fmt.Errorf("%s: %w", op, err)
	}

	log.Info("post is saved successfully")
	return postID, createdAt, nil
}

// GetPostByID возвращает пост с указанным идентификатором
// Возвращает ошибку, если не удалось получить список статей или список комментариев или
// если превышено время ожмдания ответа от БД
func (p *Post) GetPostByID(ctx context.Context, postID int64) (
	models.Post, error) {
	const op = "internal/services/posts.GetPostByID"

	log := p.Log.With(slog.String("op", op), slog.Int64("article id", postID))
	log.Info("attempting to get post by id")

	post, err := p.PostProvider.ProvidePost(ctx, postID)
	if err != nil {
		if errors.Is(err, storage.ErrPostNotFound) {
			log.Warn("post not found", sl.Err(err))
			return models.Post{}, fmt.Errorf("%s: %w", op, ErrPostNotFound)
		}

		if errors.Is(err, storage.ErrConnectionTime) {
			log.Error("connection time expired", sl.Err(err))
			return models.Post{}, fmt.Errorf("%s: %w", op, ErrConnectionTime)
		}

		if errors.Is(err, storage.ErrGetComments) {
			log.Error("could not get comments", sl.Err(err))
			return models.Post{}, fmt.Errorf("%s: %w", op, ErrGetComments)
		}

		if errors.Is(err, storage.ErrFoundComments) {
			log.Warn("could not find comments", sl.Err(err))
			return models.Post{}, fmt.Errorf("%s: %w", op, ErrFoundComments)
		}

		log.Error("failed to get post", sl.Err(err))
		return models.Post{}, fmt.Errorf("%s: %w", op, err)
	}

	log.Info("post is retrieved successfully")
	return post, nil
}

// GetAllPosts возвращает список всех статей
// Возвращает ошибку, если превышено время ожмдания ответа от БД
func (p *Post) GetAllPosts(ctx context.Context, page int64) ([]models.Post, error) {
	const op = "internal/services/posts.GetAllPosts"

	log := p.Log.With(slog.String("op", op))
	log.Info("attempting to get all posts")

	posts, err := p.PostProvider.ProvideAllPosts(ctx, page)
	if err != nil {
		if errors.Is(err, storage.ErrGetPosts) {
			log.Error("posts not found", sl.Err(err))
			return []models.Post{}, ErrGetPosts
		}

		if errors.Is(err, storage.ErrConnectionTime) {
			log.Error("connection time expired", sl.Err(err))
			return []models.Post{}, fmt.Errorf("%s: %w", op, ErrConnectionTime)
		}

		log.Error("failed to fetch all posts", sl.Err(err))
		return []models.Post{}, fmt.Errorf("%s: %w", op, err)
	}

	log.Info("posts retrieved successfully")
	return posts, nil
}