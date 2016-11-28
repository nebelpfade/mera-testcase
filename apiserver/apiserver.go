package main

import (
	"database/sql"
	"encoding/json"
	"github.com/gorilla/mux"
	_ "github.com/lib/pq"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"
)

//Global to be available anywhere
var db *sql.DB

//Structure to convert data to json and back
type ServerUptime struct {
	FQDN   string  `json:"server_fqdn"`
	Uptime float32 `json:"uptime"`
}

//Array of servers to process GET requests
type Servers []ServerUptime

//Process GET requests
func GetEndpoint(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	var (
		err   error
		host  ServerUptime
		count float32
		resp  Servers
	)
	//Just check if there any data exist
	err = db.QueryRow(`SELECT COUNT(*) FROM uptime`).Scan(&count)
	if err != nil {
		if err == sql.ErrNoRows {
			log.Println("no rows found for processing")
			w.WriteHeader(200)
			w.Write([]byte(`[]`))
			return
		} else {
			log.Println(err)
			w.WriteHeader(500)
			return
		}
	}
	rows, err := db.Query(`SELECT DISTINCT ON (fqdn) fqdn FROM uptime ` +
		`WHERE NOW() > stamp::timestamp AND ` +
		`NOW() - stamp::timestamp <= interval '24 hours'`)
	if err != nil {
		log.Println(err)
		w.WriteHeader(500)
		return
	}
	defer rows.Close()
	stmt, err := db.Prepare("SELECT COUNT(fqdn) FROM uptime " +
		"WHERE fqdn=$1 AND NOW() > stamp::timestamp AND " +
		`NOW() - stamp::timestamp <= interval '24 hours'`)
	if err != nil {
		log.Println(err)
		w.WriteHeader(500)
		return
	}
	for rows.Next() {
		err = rows.Scan(&host.FQDN)
		if err != nil {
			log.Println(err)
			w.WriteHeader(500)
			return
		}
		err := stmt.QueryRow(host.FQDN).Scan(&count)
		if err != nil {
			log.Println(err)
			continue
		}
		resp = append(resp, ServerUptime{FQDN: host.FQDN, Uptime: count/(60*24)})
	}
	err = rows.Err()
	if err != nil {
		log.Println(err)
	}
	json.NewEncoder(w).Encode(resp)
}

//Process POST requests
//Here we save request body to variable for using it in logs
func PostEndpoint(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	var (
		fqdn ServerUptime
		err  error
	)
	body, err := ioutil.ReadAll(r.Body) //read body to variable
	if err != nil {
		log.Println(err)
		w.WriteHeader(400)
		return
	}
	err = json.Unmarshal(body, &fqdn)
	if err != nil || fqdn.FQDN == "" { //check that json has FQDN field and valid
		log.Println("error to parse incomming package: " + string(body))
		w.WriteHeader(400)
		return
	}
	_, err = db.Exec("INSERT INTO uptime VALUES (default, $1)", fqdn.FQDN)
	if err != nil {
		log.Println(err)
		w.WriteHeader(500)
		return
	}
	log.Println("heartbeat for: " + fqdn.FQDN + " stored success")
	w.WriteHeader(200)
}

func main() {
	var (
		err    error
		dbHost string
		dBase  string
		dbUser string
		dbPass string
	)
	//Read DB connection params from env variables if exist
	if len(os.Getenv("POSTGRES_HOST")) > 0 {
		dbHost = os.Getenv("POSTGRES_HOST")
	} else {
		dbHost = "db"
	}
	if len(os.Getenv("POSTGRES_DB")) > 0 {
		dBase = os.Getenv("POSTGRES_DB")
	} else {
		dBase = "uptime"
	}
	if len(os.Getenv("POSTGRES_USER")) > 0 {
		dbUser = os.Getenv("POSTGRES_USER")
	} else {
		dbUser = "uptime"
	}
	if len(os.Getenv("POSTGRES_PASSWORD")) > 0 {
		dbPass = os.Getenv("POSTGRES_PASSWORD")
	} else {
		dbPass = "MeraUptimeTestCase"
	}
	conn := "postgres://" + dbUser + ":" + dbPass + "@" + dbHost + "/" + dBase + "?sslmode=disable"
	log.Println("prepairing db connection string...")
	db, err = sql.Open("postgres", conn)
	//If DB connection string is invalid then exit with fatal error
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()
	log.Println("done")
	//Try to connect to DB for 5 minutes (because container with DB can start and init late)
	for i := 1; i <= 30; i++ {
		log.Println("connecting to database retry: " + strconv.Itoa(i) + " of 30...")
		err = db.Ping()
		if err != nil {
			log.Println(err)
			time.Sleep(10 * time.Second)
		} else {
			log.Println("connected success")
			break
		}
	}
	//Exit if DB still unavailable after 5 minutes of retries
	if err != nil {
		log.Println("connection failed, aborting...")
		os.Exit(1)
	}
	log.Println("done")
	log.Println("prepairing database tables...")
	//Create database table if not exists
	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS ` +
		`uptime("stamp" timestamp default current_timestamp, ` +
		`"fqdn" varchar(200) not null)`)
	if err != nil {
		log.Fatal(err)
	}
	log.Println("done")
	//Start HTTP server
	router := mux.NewRouter()
	router.HandleFunc("/", GetEndpoint).Methods("GET")
	router.HandleFunc("/", PostEndpoint).Methods("POST")
	log.Println("starting HTTP server on port 80")
	log.Fatal(http.ListenAndServe(":80", router))
}

