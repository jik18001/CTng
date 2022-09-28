package fakeLogger

/*
Code ownership:
Finn - Wrote all functions
*/

import (
	"CTngv1/crypto"
	"CTngv1/gossip"
	"CTngv1/util"
	"crypto/rsa"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
	"strconv"
	"github.com/gorilla/mux"
)

// Logger configs are read from JSON files. The specify the CTngID of the signer, the port to host on, the MMD to make new sths with,
// The private key to sign sths with, and the misbehavior interval.
// note that other entities must have this private key copy-pasted in their cryptoconfigs to accept these sths.
type LoggerConfig struct {
	Signer              crypto.CTngID
	Port                string
	MMD                 int
	Private             rsa.PrivateKey
	MisbehaviorInterval int
}

type STH struct {
	Timestamp string
	RootHash  string
	TreeSize  int
}

//Caution: this file is plagued with Global Variables. This is ok for a stub, just makes it slightly harder to read.
var loggerType int
var currentPeriod int
var config LoggerConfig
var STHS []gossip.Gossip_object
var fakeSTHs []gossip.Gossip_object
var request_count int

// Generates a fake STH and returns a gossip object of that STH.
func generateSTH(loggerType int, Period_num int) gossip.Gossip_object {
	// Generate a random-ish STH, add to STHS.
	hashmsg := "Root Hash" + fmt.Sprint(currentPeriod+request_count)
	hash, _ := crypto.GenerateSHA256([]byte(hashmsg))
	STH1 := STH{
		Timestamp: gossip.GetCurrentTimestamp(),
		RootHash:  hex.EncodeToString(hash),
		TreeSize:  currentPeriod * 12571285,
	}
	payload0 := string(config.Signer)
	sth_payload, _ := json.Marshal(STH1)
	payload1 := string(sth_payload)
	payload2 := ""
	signature, _ := crypto.RSASign([]byte(payload0+payload1+payload2), &config.Private, config.Signer)
	gossipSTH := gossip.Gossip_object{
		Application: "CTng",
		Type:        gossip.STH,
		Period:      strconv.Itoa(Period_num),
		Signer:      string(config.Signer),
		Timestamp:   STH1.Timestamp,
		Signature:   [2]string{signature.String(), ""},
		Crypto_Scheme: "RSA",
		Payload:     [3]string{payload0, payload1, payload2},
	}
	return gossipSTH
}

// Tasks that are run each MMD:
// - Creates 2 STHs
// increments currentPeriod counter for tracking misbehaviorIntervals.
func periodicTasks() {
	// Queue the next tasks to occur at next MMD.
	time.AfterFunc(time.Duration(config.MMD)*time.Second, periodicTasks)
	// Generate STH and FakeSTH
	cperiod := gossip.GetCurrentPeriod()
	fmt.Println("Logger Running Tasks at Period ", cperiod)
	/*
	sth1 := generateSTH(loggerType)
	request_count++
	fakeSTH1 := generateSTH(loggerType)
	STHS = append(STHS, sth1)
	fakeSTHs = append(fakeSTHs, fakeSTH1)
	*/
	currentPeriod++
}
func fill_with_data(){
	STHS = STHS[:0]
	fakeSTHs = fakeSTHs[:0]
	for i:=0; i<60; i++{
		sth1 := generateSTH(loggerType,i)
		fakeSTH1 := generateSTH(loggerType,i)
		STHS = append(STHS, sth1)
	    fakeSTHs = append(fakeSTHs, fakeSTH1)
	}
}
func requestSTH(w http.ResponseWriter, r *http.Request){
	STH_index,err := strconv.Atoi(gossip.GetCurrentPeriod())
	if err == nil{}
	json.NewEncoder(w).Encode(STHS[STH_index])
}

//func requestSTH(w http.ResponseWriter, r *http.Request) {
	/*
	var PeriodByte []byte
	json.NewDecoder(r.Body).Decode(&PeriodByte)
	Periodstring := string(PeriodByte)
	reqPeriod, err := strconv.Atoi(Periodstring)
	if err != nil{
		return
	}
	*/
	/*
	//Disconnecting logger:
	request_count++
	if loggerType == 3 && currentPeriod%config.MisbehaviorInterval == 0 {
		// No response or any bad request response should trigger the accusation
		return
	}
	// Split-World Logger
	if loggerType == 2 && request_count%2 == 0 && currentPeriod%config.MisbehaviorInterval == 0{
		json.NewEncoder(w).Encode(fakeSTHs[currentPeriod-1])
		return
	}
	// Normal logger
	json.NewEncoder(w).Encode(STHS[currentPeriod-1])
}*/



// Prompts used and accepts input from the user.
// If something other than a 1,2, or 3, are printed, it is treated as a 1.
func getLoggerType() {
	fmt.Println("What type of Logger would you like to use?")
	fmt.Println("1. Normal, behaving Logger (default)")
	fmt.Println("2. Split-World (Two different STHS on every", config.MisbehaviorInterval, "MMD)")
	fmt.Println("3. Disconnecting Logger (unresponsive every", config.MisbehaviorInterval, "MMD)")
	fmt.Scanln(&loggerType)
}

// Runs a fake logger server with the ability to act roguely.
// Note that the monitor configurations must include the fakeLogger's Public key and ID as trusted, which
// Requires copying them from the fakelogger config file that is being used. (see testData/fakeLogger/logger1.json)
// This is run by the main entrypoint of the application.
func RunFakeLogger(configFile string) {
	// Global Variable initialization
	loggerType = 1
	currentPeriod = 0
	request_count = 0
	STHS = make([]gossip.Gossip_object, 0, 60)
	fakeSTHs = make([]gossip.Gossip_object, 0, 60)
	// Read the config file to the struct
	config = LoggerConfig{}
	configBytes, err := util.ReadByte(configFile)
	if err != nil {
		fmt.Println("Error reading config file: ", err)
		return
	}
	err = json.Unmarshal(configBytes, &config)
	if err != nil {
		fmt.Println("Error reading config file: ", err)
	}
	// request the object type from the user
	//getLoggerType()
	// MUX which routes HTTP directories to functions.
	gorillaRouter := mux.NewRouter().StrictSlash(true)
	// because we use global variables, we dont need to bind anything to requestSTH like we do for the other files.
	gorillaRouter.HandleFunc("/ctng/v2/get-sth", requestSTH).Methods("GET")
	http.Handle("/", gorillaRouter)
	/*
	gorillaRouter := mux.NewRouter()
	gorillaRouter.HandleFunc("/ctng/v2/get-sth/{Period}",requestSTH)
	http.Handle("/", gorillaRouter)
	*/
	fmt.Println("Listening on port", config.Port)
	fill_with_data()
	// start the server for editing STHs and serve the STHs
	go periodicTasks()
	http.ListenAndServe(":"+config.Port, nil)
}