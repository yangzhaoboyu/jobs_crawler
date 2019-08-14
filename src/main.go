package main

import (
	"encoding/json"
	"fmt"
	"github.com/PuerkitoBio/goquery"
	"github.com/gocolly/colly"
	"github.com/tealeg/xlsx"
	"log"
	"net/url"
	"strings"
	"time"
)

type Career struct {
	Name             string            //职业名称
	AvgSalary        string            //平均工资
	Experience       map[string]string //工作年限薪资
	City             map[string]string //地区薪资排行
	EducationDemand  map[string]string //学历要求
	ExperienceDemand map[string]string //经验要求
	CityDemand       map[string]string //招聘需求排行
}

const (
	homeUrl   string = "https://www.jobui.com/job?cityKw=%E5%85%A8%E5%9B%BD" //首页地址
	salaryUrl string = "https://www.jobui.com/salary/%s"                     //薪资收入地址
	trendsUrl string = "https://www.jobui.com/trends/%s"                     //就业形势地址
)

func main() {

	//工资收入水平 https://www.jobui.com/salary/全国-销售经理/
	//就业形势 https://www.jobui.com/trends/全国-销售经理/

	careerNames := make([]string, 0, 1)
	careers := make(map[string]Career)
	job := colly.NewCollector()
	job.OnHTML(".job-list-box", func(element *colly.HTMLElement) {
		element.DOM.Find(".j-work-detail-list a").Each(func(i int, selection *goquery.Selection) {
			if selection.Text() != "" {
				careerNames = append(careerNames, selection.Text())
			}
		})
	})
	job.Visit(homeUrl)

	salary := colly.NewCollector()
	salary.Limit(&colly.LimitRule{
		DomainGlob:  "*",
		Parallelism: 2,
		Delay:       500 * time.Millisecond,
	})

	trends := colly.NewCollector()
	trends.Limit(&colly.LimitRule{
		DomainGlob:  "*",
		Parallelism: 2,
		Delay:       500 * time.Millisecond,
	})

	//薪资收入解析
	salary.OnHTML("body", func(element *colly.HTMLElement) {
		name, _ := element.DOM.Find("#jobKw").Attr("value")

		//行业
		industry := make(map[string]string)
		element.DOM.Find(".fl ul li").Each(func(i int, selection *goquery.Selection) {
			industry[strings.TrimSpace(selection.Find(".industry-sample").Text())] = strings.TrimSpace(selection.Find(".cfix .money").Text())
		})

		//地区
		city := make(map[string]string)
		element.DOM.Find(".fr ul li").Each(func(i int, selection *goquery.Selection) {
			city[strings.TrimSpace(selection.Find(".areaCont").Text())] = strings.TrimSpace(selection.Find(".cfix .money").Text())
		})

		//薪资及工作年限
		exper := make(map[string]string)
		s := strings.TrimSpace(element.DOM.Find("#workLen-dataInfoss").Text())
		if s == "" {
			return
		}
		target := s[strings.Index(s, "其中")+6 : strings.LastIndex(s, "，")]
		speeds := strings.Split(target, "，")
		for _, speed := range speeds {
			ex := strings.Split(speed, "工资")
			exper[ex[0]] = ex[1]
		}

		career := Career{
			Name:       name,
			AvgSalary:  strings.TrimSpace(element.DOM.Find(".salary-avger").Text()),
			Experience: exper,
			City:       city,
		}
		careers[name] = career
	})

	//就业形势解析
	trends.OnHTML("body", func(element *colly.HTMLElement) {
		name, _ := element.DOM.Find("#sweeper").Attr("value")
		careen := careers[name]
		careen.Name = name
		element.DOM.Find(".hori3p1").Each(func(i int, hori *goquery.Selection) {
			experience := make(map[string]string)
			if hori.Find("h3").Text() == "经验要求" {
				hori.Find("li").Each(func(i int, selection *goquery.Selection) {
					key, _ := selection.Find("dfn").Attr("title")
					value := selection.Find("em").Last().Text()
					experience[key] = value
				})
				careen.ExperienceDemand = experience
			}
			education := make(map[string]string)
			if hori.Find("h3").Text() == "学历要求" {
				hori.Find("li").Each(func(i int, selection *goquery.Selection) {
					key, _ := selection.Find("dfn").Attr("title")
					value := selection.Find("em").Last().Text()
					education[key] = value
				})
				careen.EducationDemand = education
			}
		})

		//招聘地区
		cityDemand := make(map[string]string)
		element.DOM.Find(".modBar .br ol li").Each(func(i int, selection *goquery.Selection) {
			key := selection.Find(".barlist-title a").Text()
			value := selection.Find(".barlist-value em").Text()
			cityDemand[key] = value
		})
		careen.CityDemand = cityDemand
		careers[name] = careen
	})

	//持久化至Xml
	file := xlsx.NewFile()
	sheet, err := file.AddSheet("Sheet1")
	if err != nil {
		log.Fatal(err)
	}
	var row *xlsx.Row
	var cell *xlsx.Cell
	for _, career := range careerNames {

		salaryUrl := fmt.Sprintf(salaryUrl, url.QueryEscape(fmt.Sprintf("全国-%s", career)))
		salary.Visit(salaryUrl)

		trendsUrl := fmt.Sprintf(trendsUrl, url.QueryEscape(fmt.Sprintf("全国-%s", career)))
		trends.Visit(trendsUrl)

		fmt.Println(career)

		if careerInfo, ok := careers[career]; ok {
			row = sheet.AddRow()
			cell = row.AddCell()
			cell.Value = careerInfo.Name

			cell = row.AddCell()
			cell.Value = careerInfo.AvgSalary

			experienceJson, _ := json.Marshal(careerInfo.Experience)
			cell = row.AddCell()
			cell.Value = string(experienceJson)

			cityJson, _ := json.Marshal(careerInfo.City)
			cell = row.AddCell()
			cell.Value = string(cityJson)

			educationDemandJson, _ := json.Marshal(careerInfo.EducationDemand)
			cell = row.AddCell()
			cell.Value = string(educationDemandJson)

			experienceDemandJson, _ := json.Marshal(careerInfo.ExperienceDemand)
			cell = row.AddCell()
			cell.Value = string(experienceDemandJson)

			cityDemandJson, _ := json.Marshal(careerInfo.CityDemand)
			cell = row.AddCell()
			cell.Value = string(cityDemandJson)

			err = file.Save("test_write.xlsx")
			if err != nil {
				fmt.Printf(err.Error())
			}
		}
	}

}
