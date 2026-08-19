package main

import (
	"encoding/json"
	"net/http"
	"strconv"
	"slices"

	"github.com/gorilla/mux"
)

type Director struct{
	Firstname string `json:"firstname"`
	Lastname string `json:"lastname"`
}

type Movie struct{
	ID string `json:"id"`
	Isbin string `json:"isbn"`
	Title string `json:"title"`
	Director Director `json:"director"`
}

var movies []Movie
var allMovieIDs []int

func getMovies(w http.ResponseWriter, r *http.Request){
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(movies)
}

func getMovie(w http.ResponseWriter, r *http.Request){
	w.Header().Set("Content-Type", "application/json")
	params := mux.Vars(r)
	for _, item := range movies{
		if item.ID == params["id"]{
			json.NewEncoder(w).Encode(item)
			return
		}
		
	}
}

func deleteMovie(w http.ResponseWriter, r *http.Request){
	w.Header().Set("Content-Type", "application/json")
	params := mux.Vars(r)
	for index, item := range movies{
		if item.ID == params["id"]{
			movies = append(movies[:index], movies[:index+1]...)
			return
		}
	}
	json.NewEncoder(w).Encode(movies)
}

func createMovie(w http.ResponseWriter, r *http.Request){
	w.Header().Set("Content-Type", "application/json")
	var newMovie Movie
	slices.Sort(allMovieIDs[:])
	json.NewDecoder(r.Body).Decode(&newMovie)
	if len(allMovieIDs) == 0{
		allMovieIDs = append(allMovieIDs, 1)
		newMovie.ID = "1"
		movies = append(movies,newMovie)
		json.NewEncoder(w).Encode(movies)
		return 
	}
	lastMovieID := allMovieIDs[len(allMovieIDs)-1]
	allMovieIDs = append(allMovieIDs,lastMovieID + 1 )
	newMovie.ID = strconv.Itoa(lastMovieID + 1)
	movies = append(movies, newMovie)
	json.NewEncoder(w).Encode(movies)
	return
}

func updateMovie(w http.ResponseWriter, r *http.Request){
	w.Header().Set("Cotent-Type", "application/json")
	params := mux.Vars(r)
	var newMovie Movie
	json.NewDecoder(r.Body).Decode(&newMovie)
	for index, item := range movies{
		if item.ID == params["id"]{
			movies = append(movies[:index],movies[index+1:]...)
			movies = append(movies, newMovie)
			parsedInt, _ := strconv.Atoi(newMovie.ID)
			allMovieIDs = append(allMovieIDs, parsedInt)
			json.NewEncoder(w).Encode(movies)
			return
		}
	}
	
}

func main(){
	r := mux.NewRouter()
	movie1 := &Movie{ID:"1", Isbin: "456", Title: "Dark Knight", Director: Director{Firstname: "James", Lastname: "Tang"} }
	movie2 := &Movie{ID:"2", Isbin: "678", Title: "Kung Fu Panda", Director: Director{Firstname: "Justin", Lastname: "Chen"}}
	movies = append(movies, *movie1, *movie2)
	allMovieIDs = append(allMovieIDs, 1,2)
	r.HandleFunc("/movies", getMovies).Methods("GET")
	r.HandleFunc("/movies/{id}", getMovie).Methods("GET")
	r.HandleFunc("/movies", createMovie).Methods("Post")
	r.HandleFunc("/movies/{id}", updateMovie).Methods("PUT")
	r.HandleFunc("/movies/{id}", deleteMovie).Methods("DELETE")

}