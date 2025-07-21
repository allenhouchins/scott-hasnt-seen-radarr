package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/joho/godotenv"
)

// Movie represents a movie with its metadata
type Movie struct {
	Title     string `json:"title"`
	IMDBID    string `json:"imdb_id"`
	PosterURL string `json:"poster_url"`
}

// TMDBResponse represents the response from TMDB API
type TMDBResponse struct {
	Results []TMDBMovie `json:"results"`
}

// TMDBMovie represents a movie from TMDB API
type TMDBMovie struct {
	ID          int    `json:"id"`
	Title       string `json:"title"`
	PosterPath  string `json:"poster_path"`
	ReleaseDate string `json:"release_date"`
	GenreIDs    []int  `json:"genre_ids"`
}

// TMDBExternalIDs represents external IDs from TMDB API
type TMDBExternalIDs struct {
	IMDBID string `json:"imdb_id"`
}

// Genre mapping from TMDB genre IDs to names
var genreMap = map[int]string{
	28:    "action",
	12:    "adventure",
	16:    "animation",
	35:    "comedy",
	80:    "crime",
	99:    "documentary",
	18:    "drama",
	10751: "family",
	14:    "fantasy",
	36:    "history",
	27:    "horror",
	10402: "music",
	9648:  "mystery",
	10749: "romance",
	878:   "science_fiction",
	10770: "tv_movie",
	53:    "thriller",
	10752: "war",
	37:    "western",
}

// Scraper handles the scraping and API interactions
type Scraper struct {
	tmdbAPIKey string
	client     *http.Client
	wikiURL    string
	tmdbBaseURL string
}

// NewScraper creates a new scraper instance
func NewScraper(apiKey string) *Scraper {
	return &Scraper{
		tmdbAPIKey:  apiKey,
		client:      &http.Client{Timeout: 30 * time.Second},
		wikiURL:     "https://comedybangbang.fandom.com/wiki/Scott_Hasn%27t_Seen",
		tmdbBaseURL: "https://api.themoviedb.org/3",
	}
}

// scrapeWikiPage fetches the Scott Hasn't Seen wiki page
func (s *Scraper) scrapeWikiPage() (string, error) {
	resp, err := s.client.Get(s.wikiURL)
	if err != nil {
		return "", fmt.Errorf("failed to fetch wiki page: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("wiki page returned status: %d", resp.StatusCode)
	}

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to parse HTML: %w", err)
	}

	return doc.Html()
}

// extractMovieTitles extracts movie titles from the HTML content
func (s *Scraper) extractMovieTitles(htmlContent string) ([]string, error) {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(htmlContent))
	if err != nil {
		return nil, fmt.Errorf("failed to parse HTML: %w", err)
	}

	var movies []string
	seen := make(map[string]bool)

	// Find all italicized text (movie titles)
	doc.Find("i").Each(func(i int, s *goquery.Selection) {
		title := strings.TrimSpace(s.Text())
		
		// Skip if already seen
		if seen[title] {
			return
		}
		seen[title] = true

		// Skip very short titles
		if len(title) < 3 {
			return
		}

		// Skip non-movie entries
		skipKeywords := []string{
			"cobra kai", "season", "episodes", "pilot", "watchalong",
			"awards", "the scott hasn't seenies", "march of the penguins",
			"september 5", "twin peaks", "martin", "sprague hasn't seen",
			"did", "next", "the scott hasn't seenies awards",
			"scott hasn't seen", // Add the podcast name itself
		}

		titleLower := strings.ToLower(title)
		for _, keyword := range skipKeywords {
			if strings.Contains(titleLower, keyword) {
				return
			}
		}

		// Skip if contains episode/season patterns
		episodePattern := regexp.MustCompile(`(?i)episode|season|part \d+`)
		if episodePattern.MatchString(title) {
			return
		}

		// Skip single words that are too short
		words := strings.Fields(title)
		if len(words) <= 1 && len(title) < 4 {
			return
		}

		movies = append(movies, title)
	})

	return movies, nil
}

// searchMovie searches for a movie on TMDB
func (s *Scraper) searchMovie(title string) (*Movie, error) {
	// Handle special cases with "/" in titles
	if strings.Contains(title, "/") {
		// Try the full title first
		movie, err := s.searchMovieExact(title)
		if err == nil {
			return movie, nil
		}
		
		// If that fails, try splitting by "/" and search for the first part
		parts := strings.Split(title, "/")
		if len(parts) > 0 {
			firstPart := strings.TrimSpace(parts[0])
			if firstPart != "" {
				movie, err := s.searchMovieExact(firstPart)
				if err == nil {
					return movie, nil
				}
			}
		}
		
		// If splitting fails, return the original error
		return nil, fmt.Errorf("no results found for '%s' (tried full title and first part)", title)
	}
	
	return s.searchMovieExact(title)
}

// searchMovieExact searches for a movie on TMDB with exact title
func (s *Scraper) searchMovieExact(title string) (*Movie, error) {
	searchURL := fmt.Sprintf("%s/search/movie", s.tmdbBaseURL)
	
	params := url.Values{}
	params.Add("api_key", s.tmdbAPIKey)
	params.Add("query", title)
	params.Add("language", "en-US")
	params.Add("page", "1")
	params.Add("include_adult", "false")

	req, err := http.NewRequest("GET", searchURL+"?"+params.Encode(), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to search movie '%s': %w", title, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("TMDB API returned status %d for '%s'", resp.StatusCode, title)
	}

	var tmdbResp TMDBResponse
	if err := json.NewDecoder(resp.Body).Decode(&tmdbResp); err != nil {
		return nil, fmt.Errorf("failed to decode TMDB response: %w", err)
	}

	if len(tmdbResp.Results) == 0 {
		return nil, fmt.Errorf("no results found for '%s'", title)
	}

	movie := tmdbResp.Results[0]
	
	// Get IMDB ID
	imdbID, err := s.getIMDBID(movie.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to get IMDB ID for '%s': %w", title, err)
	}

	posterURL := ""
	if movie.PosterPath != "" {
		posterURL = fmt.Sprintf("https://www.themoviedb.org/t/p/w300_and_h450_bestv2%s", movie.PosterPath)
	}

	return &Movie{
		Title:     movie.Title,
		IMDBID:    imdbID,
		PosterURL: posterURL,
	}, nil
}

// getGenres converts genre IDs to genre names
func (s *Scraper) getGenres(genreIDs []int) []string {
	var genres []string
	for _, id := range genreIDs {
		if genreName, exists := genreMap[id]; exists {
			genres = append(genres, genreName)
		}
	}
	return genres
}

// getIMDBID gets the IMDB ID for a TMDB movie ID
func (s *Scraper) getIMDBID(tmdbID int) (string, error) {
	apiURL := fmt.Sprintf("%s/movie/%d/external_ids", s.tmdbBaseURL, tmdbID)
	
	params := url.Values{}
	params.Add("api_key", s.tmdbAPIKey)

	req, err := http.NewRequest("GET", apiURL+"?"+params.Encode(), nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := s.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to get external IDs: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("TMDB API returned status %d for external IDs", resp.StatusCode)
	}

	var externalIDs TMDBExternalIDs
	if err := json.NewDecoder(resp.Body).Decode(&externalIDs); err != nil {
		return "", fmt.Errorf("failed to decode external IDs response: %w", err)
	}

	return externalIDs.IMDBID, nil
}

// generateRadarrList generates the complete Radarr-compatible list
func (s *Scraper) generateRadarrList() ([]Movie, error) {
	fmt.Println("Scraping Scott Hasn't Seen wiki page...")
	htmlContent, err := s.scrapeWikiPage()
	if err != nil {
		return nil, fmt.Errorf("failed to scrape wiki page: %w", err)
	}

	fmt.Println("Extracting movie titles...")
	movieTitles, err := s.extractMovieTitles(htmlContent)
	if err != nil {
		return nil, fmt.Errorf("failed to extract movie titles: %w", err)
	}

	fmt.Printf("Found %d unique movies\n", len(movieTitles))

	var radarrList []Movie
	var mu sync.Mutex
	var wg sync.WaitGroup

	// Use a semaphore to limit concurrent API calls
	semaphore := make(chan struct{}, 5) // Limit to 5 concurrent requests

	successful := 0
	failed := 0

	for i, title := range movieTitles {
		wg.Add(1)
		go func(index int, movieTitle string) {
			defer wg.Done()
			
			// Acquire semaphore
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			fmt.Printf("Processing %d/%d: %s\n", index+1, len(movieTitles), movieTitle)

			movie, err := s.searchMovie(movieTitle)
			if err != nil {
				mu.Lock()
				failed++
				mu.Unlock()
				fmt.Printf("  ✗ Not found: %s (%v)\n", movieTitle, err)
				return
			}

			// Only require IMDB ID (essential for Radarr), poster URL is optional
			if movie.IMDBID != "" {
				mu.Lock()
				radarrList = append(radarrList, *movie)
				successful++
				mu.Unlock()
				
				// Log whether poster is available or not
				if movie.PosterURL != "" {
					fmt.Printf("  ✓ Found: %s (IMDB: %s)\n", movie.Title, movie.IMDBID)
				} else {
					fmt.Printf("  ✓ Found: %s (IMDB: %s) - No poster\n", movie.Title, movie.IMDBID)
				}
			} else {
				mu.Lock()
				failed++
				mu.Unlock()
				fmt.Printf("  ✗ Missing IMDB ID: %s\n", movieTitle)
			}

			// Rate limiting
			time.Sleep(250 * time.Millisecond)
		}(i, title)
	}

	wg.Wait()

	// Sort the movies by title to ensure consistent order
	sort.Slice(radarrList, func(i, j int) bool {
		return radarrList[i].Title < radarrList[j].Title
	})
	
	fmt.Println("Movies sorted by title for consistent output order")

	fmt.Printf("\nSummary:\n")
	fmt.Printf("  Successful: %d\n", successful)
	fmt.Printf("  Failed: %d\n", failed)
	fmt.Printf("  Total: %d\n", len(radarrList))

	return radarrList, nil
}

// saveToFile saves the Radarr list to a JSON file
func (s *Scraper) saveToFile(movies []Movie, filename string) error {
	data, err := json.Marshal(movies)
	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %w", err)
	}

	// Add newline at the end of the JSON data
	data = append(data, '\n')

	if err := os.WriteFile(filename, data, 0644); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	fmt.Printf("Saved %d movies to %s\n", len(movies), filename)
	return nil
}

func main() {
	// Load environment variables from .env file if it exists
	godotenv.Load()

	// Get TMDB API key from environment
	tmdbAPIKey := os.Getenv("TMDB_API_KEY")
	if tmdbAPIKey == "" {
		log.Fatal("Error: TMDB_API_KEY environment variable not set\nPlease get your API key from https://www.themoviedb.org/settings/api")
	}

	scraper := NewScraper(tmdbAPIKey)
	radarrList, err := scraper.generateRadarrList()
	if err != nil {
		log.Fatalf("Failed to generate Radarr list: %v", err)
	}

	if len(radarrList) > 0 {
		// Debug: Show current working directory
		if cwd, err := os.Getwd(); err == nil {
			fmt.Printf("Current working directory: %s\n", cwd)
		}
		
		// Save with timestamp
		timestamp := time.Now().Format("20060102_150405")
		filename := fmt.Sprintf("../../scott_hasnt_seen_%s.json", timestamp)
		fmt.Printf("Saving timestamped file to: %s\n", filename)
		if err := scraper.saveToFile(radarrList, filename); err != nil {
			log.Printf("Failed to save timestamped file: %v", err)
		}

		// Save without timestamp for easy access (in root directory)
		mainFilename := "../../scott_hasnt_seen.json"
		fmt.Printf("Saving main file to: %s\n", mainFilename)
		if err := scraper.saveToFile(radarrList, mainFilename); err != nil {
			log.Printf("Failed to save main file: %v", err)
		}
	} else {
		fmt.Println("No movies found to save")
	}
} 