package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

const measurementInterval = 15 * time.Minute

func fileName() string {
	return strings.ToLower("temps" + "-" + time.Now().Month().String() + "-" + strconv.Itoa(time.Now().Year()))
}

func main() {
	tempMonitor()

	http.HandleFunc("/", func(w http.ResponseWriter, req *http.Request) {
		http.RedirectHandler("https://kaff.se", 404)
	})
	http.HandleFunc("/temp", tempHandler)
	http.HandleFunc("/temp/all", allTempsHandler)

	srv := &http.Server{
		Addr:         "127.0.0.1:35000",
		WriteTimeout: 15 * time.Second,
		ReadTimeout:  15 * time.Second,
	}
	fmt.Println("Measuring each", measurementInterval, "\nListening on:", srv.Addr)
	// Log and run the server
	log.Fatal(srv.ListenAndServe())
}

func tempHandler(w http.ResponseWriter, r *http.Request) {
	t, err := readTemp()
	if err != nil {
		w.WriteHeader(500)
		w.Write([]byte(err.Error()))
	}
	j, err := json.MarshalIndent(t, "", " ")

	if err != nil {
		w.WriteHeader(500)
		w.Write([]byte(err.Error()))
	}
	w.Header().Set("Content-Type", "application/json")
	w.Write(j)
}
func allTempsHandler(w http.ResponseWriter, r *http.Request) {
	data, err := ioutil.ReadFile(fileName())
	if err != nil {
		w.WriteHeader(500)
		w.Write([]byte(err.Error()))
	}
	w.Header().Set("Content-Type", "application/json")
	w.Write(data)
}

type reading struct {
	Deg  float64   `json:"degrees"`
	Time time.Time `json:"time"`
}

func (r *reading) string() string {
	return strconv.FormatFloat(r.Deg, 'f', 1, 32) + "⁰C"
}

type temperatures struct {
	Readings []reading `json:"Readings"`
}

func (t *temperatures) save() error {
	j, err := json.MarshalIndent(t, "", " ")
	if err != nil {
		return err
	}
	err = ioutil.WriteFile(fileName(), j, os.FileMode(0777))
	if err != nil {
		return err
	}
	return nil
}

func readStringTemp() (string, error) {
	out, err := exec.Command("/opt/vc/bin/vcgencmd", "measure_temp").Output()
	if err != nil {
		return "", err
	}
	return string(out), nil
}
func readTemp() (reading, error) {
	out, err := exec.Command("/opt/vc/bin/vcgencmd", "measure_temp").Output()
	if err != nil {
		return reading{}, err
	}
	tstr := string(out)[5:7] + "." + string(out)[8:9]
	t, err := strconv.ParseFloat(tstr, 64)
	if err != nil {
		return reading{}, err
	}
	tmp := reading{
		t,
		time.Now(),
	}

	return tmp, nil
}
func loadTemps() (temperatures, error) {

	data, err := ioutil.ReadFile(fileName())
	if err != nil {
		if data == nil {
			fmt.Println("No previous file, creating a new file:", fileName())
			ioutil.WriteFile(fileName(), nil, os.FileMode(0777))
		} else {
			return temperatures{}, err
		}
	}

	temps := temperatures{}

	err = json.Unmarshal(data, &temps)

	if err != nil {
		if data != nil {
			return temperatures{}, err
		}
	}

	return temps, nil
}
func tempMonitor() error {
	tmps, err := loadTemps()
	if err != nil {
		fmt.Println(err)
		return err
	}
	read, err := readTemp()
	if err != nil {
		fmt.Println(err)
		return err
	}
	tmps.Readings = append(tmps.Readings, read)
	t := time.NewTicker(measurementInterval)
	quit := make(chan struct{})
	go func() {
		for {
			select {
			case <-t.C:
				r, err := readTemp()
				if err != nil {
					return
				}
				fmt.Println("New reading:", r.Deg)
				tmps.Readings = append(tmps.Readings, r)
				tmps.save()
			case <-quit:
				t.Stop()
				return
			}
		}
	}()
	return nil
}
