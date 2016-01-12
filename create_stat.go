package main

import (
"fmt"
"os"
"net/http"
"io/ioutil"
"encoding/xml"
"encoding/csv"
)

type E2ETemporalExtent struct {
	XMLName xml.Name `xml:"E2ETemporalExtent"`
	BeginDateTime string `xml:"beginDateTime"`
	EndDateTime string `xml:"endDateTime"`
}

type E2ESearchMD struct {
	XMLName xml.Name `xml:"E2ESearchMD"`
	TemporalExtent E2ETemporalExtent `xml:"E2ETemporalExtent"`
}

type Root struct {
	XMLName xml.Name `xml:"root"`	
	Metadata E2ESearchMD
}

func main() {
	// filename := "RU_RIHMI-WDC_1172.xml";
	addr := "http://dp.hydrometcentre.esimo.ru:8080/dpms/controller?action=getResourceCache&resourceId="
	resources := [...]string{"RU_Hydrometcentre_42", "RU_Hydrometcentre_46", "RU_Hydrometcentre_60",
							"RU_Hydrometcentre_61", "RU_Hydrometcentre_62", "RU_Hydrometcentre_63",
							"RU_Hydrometcentre_64", "RU_Hydrometcentre_65", "RU_Hydrometcentre_66"}
	csvfile, err_csv := os.Create("ir_stat.csv")
	if err_csv != nil {
		panic("Error creating CSV file")
	}
	defer csvfile.Close()
	writer := csv.NewWriter(csvfile)
	writer.Write([]string{"sep=,"})

	for _,resource := range resources {
		fmt.Println(resource);
		writer.Write([]string{resource, ""})

		res, err := http.Get(addr + resource)
		if err != nil {
			panic(err.Error())
		}	

		body, err := ioutil.ReadAll(res.Body)

		var root Root
		err_parse := xml.Unmarshal([]byte(body), &root)
		if err_parse != nil {
			fmt.Printf("error: %root", err_parse)
		}
	
		beginDateTime := root.Metadata.TemporalExtent.BeginDateTime
		endDateTime := root.Metadata.TemporalExtent.EndDateTime

		fmt.Println(beginDateTime + " - " + endDateTime)
		writer.Write([]string{"metadata", beginDateTime + " - " + endDateTime})
		writer.Write([]string{"data", ""})
	}

	writer.Flush()
}