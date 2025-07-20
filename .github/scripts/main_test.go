package main

import (
	"fmt"
	"testing"
)

func TestExtractMovieTitles(t *testing.T) {
	scraper := NewScraper("dummy_key")
	
	// Sample HTML content with movie titles
	htmlContent := `
	<html>
		<body>
			<table>
				<tr><td><i>Space Jam</i></td></tr>
				<tr><td><i>The Addams Family</i></td></tr>
				<tr><td><i>Dune</i></td></tr>
				<tr><td><i>Ghost</i></td></tr>
				<tr><td><i>Sister Act</i></td></tr>
				<tr><td><i>Cobra Kai Season 5</i></td></tr>
				<tr><td><i>Did</i></td></tr>
				<tr><td><i>Sprague Hasn't Seen</i></td></tr>
			</table>
		</body>
	</html>
	`
	
	movies, err := scraper.extractMovieTitles(htmlContent)
	if err != nil {
		t.Fatalf("Failed to extract movie titles: %v", err)
	}
	
	// Check that we found the expected movies
	expectedMovies := []string{"Space Jam", "The Addams Family", "Dune", "Ghost", "Sister Act"}
	foundCount := 0
	
	for _, expected := range expectedMovies {
		for _, found := range movies {
			if found == expected {
				foundCount++
				break
			}
		}
	}
	
	if foundCount != len(expectedMovies) {
		t.Errorf("Expected to find %d movies, but found %d", len(expectedMovies), foundCount)
	}
	
	// Check that we filtered out non-movies
	unwantedMovies := []string{"Cobra Kai Season 5", "Did", "Sprague Hasn't Seen"}
	for _, unwanted := range unwantedMovies {
		for _, found := range movies {
			if found == unwanted {
				t.Errorf("Found unwanted movie: %s", unwanted)
			}
		}
	}
	
	t.Logf("Successfully extracted %d movies", len(movies))
}

func TestScraperCreation(t *testing.T) {
	apiKey := "test_api_key"
	scraper := NewScraper(apiKey)
	
	if scraper.tmdbAPIKey != apiKey {
		t.Errorf("Expected API key %s, got %s", apiKey, scraper.tmdbAPIKey)
	}
	
	if scraper.wikiURL != "https://comedybangbang.fandom.com/wiki/Scott_Hasn%27t_Seen" {
		t.Errorf("Unexpected wiki URL: %s", scraper.wikiURL)
	}
	
	if scraper.tmdbBaseURL != "https://api.themoviedb.org/3" {
		t.Errorf("Unexpected TMDB base URL: %s", scraper.tmdbBaseURL)
	}
}

func TestMovieStruct(t *testing.T) {
	movie := Movie{
		Title:          "Test Movie",
		PosterURL:      "https://example.com/poster.jpg",
		IMDBID:         "tt1234567",
		EpisodeNumber:  1,
		EpisodeAirDate: "2023-01-09",
	}
	
	if movie.Title != "Test Movie" {
		t.Errorf("Expected title 'Test Movie', got '%s'", movie.Title)
	}
	
	if movie.PosterURL != "https://example.com/poster.jpg" {
		t.Errorf("Expected poster URL 'https://example.com/poster.jpg', got '%s'", movie.PosterURL)
	}
	
	if movie.IMDBID != "tt1234567" {
		t.Errorf("Expected IMDB ID 'tt1234567', got '%s'", movie.IMDBID)
	}
	
	if movie.EpisodeNumber != 1 {
		t.Errorf("Expected episode number 1, got %d", movie.EpisodeNumber)
	}
	
	if movie.EpisodeAirDate != "2023-01-09" {
		t.Errorf("Expected episode air date '2023-01-09', got '%s'", movie.EpisodeAirDate)
	}
}

func TestFilteringLogic(t *testing.T) {
	scraper := NewScraper("dummy_key")
	
	testCases := []struct {
		title    string
		expected bool // true if should be included
	}{
		{"Space Jam", true},
		{"The Addams Family", true},
		{"Dune", true},
		{"Ghost", true},
		{"Cobra Kai Season 5", false},
		{"Did", false},
		{"Sprague Hasn't Seen", false},
		{"The Scott Hasn't Seenies Awards", false},
		{"Twin Peaks", false},
		{"Martin", false},
		{"", false},
		{"A", false},
		{"Ab", false},
		{"Abc", false}, // 3 characters but single word and short
	}
	
	for _, tc := range testCases {
		// Create a simple HTML with just this title
		htmlContent := fmt.Sprintf("<html><body><i>%s</i></body></html>", tc.title)
		
		movies, err := scraper.extractMovieTitles(htmlContent)
		if err != nil {
			t.Errorf("Error extracting titles for '%s': %v", tc.title, err)
			continue
		}
		
		found := len(movies) > 0
		if found != tc.expected {
			t.Errorf("Title '%s': expected %v, got %v", tc.title, tc.expected, found)
		}
	}
} 