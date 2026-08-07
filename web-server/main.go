package main

import ( 
	"fmt"
	"log"
	"net/http"
	"strconv"
	"sync"
)

var(
	counter int
	mutex sync.Mutex
)

func showHTML(w http.ResponseWriter, r *http.Request){
	http.ServeFile(w, r, r.URL.Path[1:])
}

func incrementCounter(w http.ResponseWriter, r *http.Request){
	mutex.Lock()
	counter ++
	fmt.Fprintf(w, strconv.Itoa(counter))
	mutex.Unlock()
}

func main() {
	http.Handle("/", http.FileServer(http.Dir("./static")))

	// http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
    // 	http.ServeFile(w, r, r.URL.Path[1:])
    // })

	http.HandleFunc("/increment", incrementCounter)

	http.HandleFunc("/hi", func(w http.ResponseWriter, r *http.Request) {
        fmt.Fprintf(w, "Hi")
    })

	log.Fatal(http.ListenAndServe(":8081", nil))
}