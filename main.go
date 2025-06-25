package main

import (
	"fmt"
	"html/template"
	"log"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/template/html/v2"
	"github.com/russross/blackfriday/v2"
	"gopkg.in/yaml.v2"
)

// BlogPost represents a blog post with metadata
type BlogPost struct {
	Title       string    `yaml:"title"`
	Date        time.Time `yaml:"date"`
	Author      string    `yaml:"author"`
	Description string    `yaml:"description"`
	Tags        []string  `yaml:"tags"`
	Slug        string    `yaml:"slug"`
	Content     string    `yaml:"-"`
	HTMLContent string    `yaml:"-"`
}

// BlogMetadata represents the frontmatter of a markdown file
type BlogMetadata struct {
	Title       string    `yaml:"title"`
	Date        time.Time `yaml:"date"`
	Author      string    `yaml:"author"`
	Description string    `yaml:"description"`
	Tags        []string  `yaml:"tags"`
	Slug        string    `yaml:"slug"`
}

func main() {
	// Initialize template engine
	engine := html.New("./internal/templates", ".html")
	engine.Reload(true) // Optional. Default: false

	// Add custom template function for raw HTML
	engine.AddFunc("raw", func(s interface{}) template.HTML {
		switch v := s.(type) {
		case template.HTML:
			return v
		case string:
			return template.HTML(v)
		default:
			return ""
		}
	})

	// Create fiber app
	app := fiber.New(fiber.Config{
		Views: engine,
	})

	// Static files
	app.Static("/", "./public")

	// Routes
	app.Get("/", func(c *fiber.Ctx) error {
		posts, err := getAllBlogPosts()
		if err != nil {
			return c.Status(500).SendString("Error loading blog posts")
		}
		slog.Info("Loaded posts", "count", len(posts))
		err = c.Render("index", fiber.Map{
			"Title": "DevDaze Blog",
			"Posts": posts,
		})
		if err != nil {
			slog.Error("Template render error", "error", err)
			return c.Status(500).SendString("Template render error")
		}
		return nil
	})

	app.Get("/blog/:slug", func(c *fiber.Ctx) error {
		slug := c.Params("slug")
		post, err := getBlogPost(slug)
		if err != nil {
			return c.Status(404).SendString("Blog post not found")
		}
		return c.Render("post", fiber.Map{
			"Title": post.Title,
			"Post":  post,
		})
	})

	app.Get("/blog", func(c *fiber.Ctx) error {
		posts, err := getAllBlogPosts()
		if err != nil {
			return c.Status(500).SendString("Error loading blog posts")
		}
		return c.Render("blog", fiber.Map{
			"Title": "All Blog Posts",
			"Posts": posts,
		})
	})

	log.Println("Server starting on :3000")
	log.Fatal(app.Listen(":3000"))
}

// getBlogPost loads and parses a single blog post by slug
func getBlogPost(slug string) (*BlogPost, error) {
	contentDir := "./content"
	files, err := os.ReadDir(contentDir)
	if err != nil {
		return nil, err
	}

	for _, file := range files {
		if !strings.HasSuffix(file.Name(), ".md") {
			continue
		}

		filePath := filepath.Join(contentDir, file.Name())
		content, err := os.ReadFile(filePath)
		if err != nil {
			continue
		}

		post, err := parseMarkdownFile(content)
		if err != nil {
			continue
		}

		if post.Slug == slug {
			return post, nil
		}
	}

	return nil, fmt.Errorf("blog post with slug '%s' not found", slug)
}

// getAllBlogPosts loads and parses all blog posts
func getAllBlogPosts() ([]*BlogPost, error) {
	contentDir := "./content"
	var posts []*BlogPost

	// Check if content directory exists
	if _, err := os.Stat(contentDir); os.IsNotExist(err) {
		return posts, nil // Return empty slice if directory doesn't exist
	}

	files, err := os.ReadDir(contentDir)
	if err != nil {
		return nil, err
	}

	for _, file := range files {
		if !strings.HasSuffix(file.Name(), ".md") {
			continue
		}

		filePath := filepath.Join(contentDir, file.Name())
		content, err := os.ReadFile(filePath)
		if err != nil {
			log.Printf("Error reading file %s: %v", filePath, err)
			continue
		}

		post, err := parseMarkdownFile(content)
		if err != nil {
			log.Printf("Error parsing file %s: %v", filePath, err)
			continue
		}

		posts = append(posts, post)
	}

	return posts, nil
}

// parseMarkdownFile parses a markdown file with YAML frontmatter
func parseMarkdownFile(content []byte) (*BlogPost, error) {
	contentStr := string(content)

	// Check for frontmatter
	if !strings.HasPrefix(contentStr, "---") {
		return nil, fmt.Errorf("no frontmatter found")
	}

	// Split frontmatter and content
	parts := strings.SplitN(contentStr[3:], "---", 2)
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid frontmatter format")
	}

	frontmatter := strings.TrimSpace(parts[0])
	markdownContent := strings.TrimSpace(parts[1])

	// Parse YAML frontmatter
	var metadata BlogMetadata
	err := yaml.Unmarshal([]byte(frontmatter), &metadata)
	if err != nil {
		return nil, fmt.Errorf("error parsing frontmatter: %v", err)
	}

	// Convert markdown to HTML
	htmlContent := blackfriday.Run([]byte(markdownContent))

	// Create blog post
	post := &BlogPost{
		Title:       metadata.Title,
		Date:        metadata.Date,
		Author:      metadata.Author,
		Description: metadata.Description,
		Tags:        metadata.Tags,
		Slug:        metadata.Slug,
		Content:     markdownContent,
		HTMLContent: string(htmlContent),
	}

	return post, nil
}
