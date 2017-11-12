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
	"time"
)

func main() {
	tempMonitor(1)
	http.HandleFunc("/temp", tempHandler)
	http.HandleFunc("/temp/all", allTempsHandler)

	srv := &http.Server{
		Addr:         "127.0.0.1:35000",
		WriteTimeout: 15 * time.Second,
		ReadTimeout:  15 * time.Second,
	}
	fmt.Println("Handlers set up.\nListening on:", srv.Addr)
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
	w.Write([]byte(j))
}
func allTempsHandler(w http.ResponseWriter, r *http.Request) {
	data, err := ioutil.ReadFile("temps")
	if err != nil {
		w.WriteHeader(500)
		w.Write([]byte(err.Error()))
	}
	w.Write(data)
}

type reading struct {
	deg float64
	t   time.Time
}

func (r *reading) string() string {
	return strconv.FormatFloat(r.deg, 'f', 1, 64) + "⁰C"
}

type temperatures struct {
	readings []reading
}

func (t *temperatures) save() error {
	j, err := json.MarshalIndent(t, "", " ")
	if err != nil {
		return err
	}
	err = ioutil.WriteFile("temps", j, os.FileMode(0777))
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

	t, err := strconv.ParseFloat(string(out)[5:9], 64)
	tmp := reading{
		t,
		time.Now(),
	}
	/*if err != nil {
		return reading{}, err
	}
	temps, err := loadTemps()
	if err != nil {
		return reading{}, err
	}
	allTemps := append(temps.readings, tmp)
	temps.readings = allTemps
	err = temps.save()
	if err != nil {
		return reading{}, err
	}*/
	return tmp, nil
}
func loadTemps() (temperatures, error) {
	data, err := ioutil.ReadFile("temps")
	if err != nil {
		return temperatures{}, err
	}
	temps := temperatures{}
	err = json.Unmarshal(data, &temps)
	if err != nil {
		return temperatures{}, err
	}
	return temps, nil
}
func tempMonitor(interval int) error {
	tmps, err := loadTemps()
	if err != nil {
		return err
	}
	fmt.Println("Started temperature monitoring")
	read, err := readTemp()
	if err != nil {
		return err
	}
	tmps.readings = append(tmps.readings, read)
	t := time.NewTicker(1 * time.Hour)
	quit := make(chan struct{})
	go func() {
		for {
			select {
			case <-t.C:
				r, err := readTemp()
				if err != nil {
					return
				}
				log.Println("New reading:", r.deg)
				tmps.readings = append(tmps.readings, r)
				tmps.save()
			case <-quit:
				t.Stop()
				return
			}
		}
	}()
	return nil
}
