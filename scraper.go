package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/gocolly/colly" // installation: go get -u github.com/gocolly/colly/...
)

// Iz nekog razloga vristi kad ovako deklarises pa posle definises funkciju, tkd je zakomentarisano
// Cita se: func ime(arg1 tip, arg2 tip, ...) returnTip

func main() {

	// os.Args je niz argumenata komandne linije
	if len(os.Args) < 2 {
		fmt.Println("Notice: No filter was given.")
	}

	found := false

	// Instantiate default collector
	c := colly.NewCollector(
		// TODO: (STEFAN, samoprijavljujem se za ovo) izvuci domene u poseban settings fajl pa citaj odatle
		// Ti bolje baratas sa json-om ili toml, mozda mogu i citati i domeni da se drze u jednom fajlu?

		// paralelizuj pretragu
		colly.Async(true),
		// Kad bude pratio linkove, ulazi samo u one koji imaju ove domene:
		colly.AllowedDomains(
			"www.blic.rs",
			"www.telegraf.rs",
		),
	)

	// ogranici paralelne rekvestove na x odjednom
	c.Limit(&colly.LimitRule{
		Parallelism: 5,
	})

	// da nas ne blokiraju zbog previse scrapinga
	// setting a valid User-Agent header
	c.UserAgent = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/109.0.0.0 Safari/537.36"

	seenArticles := loadSeenArticles()

	// Kad nadje <a href> na stranici, pre nego sto udje u njega (obidje), prvo odradi ovo:
	c.OnHTML("a[href]", func(e *colly.HTMLElement) {
		link := e.Attr("href")
		title := e.Attr("title")

		if checkFilter(title, os.Args[1:]) {
			if !contains(title, seenArticles) {
				found = true
				seenArticles = append(seenArticles, title)

				fmt.Println(title)
				fmt.Println(link)
				fmt.Println()
			}

			c.Visit(e.Request.AbsoluteURL(link))
		}
	})

	c.OnRequest(func(r *colly.Request) {
		// fmt.Println("Visiting", r.URL.String())
	})

	// TODO: dodati jos stranica, na foru blic/tv, blic/naslovna, itd.
	// TODO: ovo bi vec valjalo drzati u nekom json-u ili obicnom fajlu da bi se lakse dodavale stranice,
	// plus tu bi potencijalno mogao i scraper automatski da dodaje ako pronadje novu stranicu
	pagesToScrape := []string{
		"https://www.blic.rs/najnovije-vesti",
		"https://www.telegraf.rs/najnovije-vesti",
	}

	// pokreni vise scrapera koji ce paralelno obilaziti stranice
	for _, page := range pagesToScrape {
		c.Visit(page)
	}

	// prikupi sve 'niti'
	c.Wait()

	// nakon sto su svi scraperi zavrsili, sacuvaj sve pregledane stranice
	saveSeenArticles(seenArticles)

	if !found { // Nije nasao nijedan artikl u kom se pominje
		fmt.Println("Not found")
	}
}

// =============== FUNCTIONS ================

// Proverava da li string haystack (plast sena) sadrzi bilo koji string needle(iglu) iz needles niza
func checkFilter(haystack string, needles []string) bool {
	// for index, value := range array
	// Iterira po indeksu a trenutni element je value (kao for each sa tim sto iterira i po indeksu) [realno BRDA korisna stvar]
	// Posto index ne koristi, moze da se napise for _, value := range array (vristi kompajler ako ti stoji index, value a jedno ne koristis)
	for _, needle := range needles {
		if strings.Contains(strings.ToLower(haystack), strings.ToLower(needle)) {
			return true
		}
	}

	return false
}

// Ucitava procitane artikle u slice stringova (slice u golangu je vektor u c++)
func loadSeenArticles() []string {
	// nema potrebe da filename dobija preko argumenta kad uvek cuvamo u isti (uskoro cu
	// napraviti nesto sa json-om ili tako nesto slicno)
	file, err := os.Open("seen_articles.txt")
	if err != nil {
		log.Println("Error:", err)
		return []string{}
	}
	defer file.Close()

	var seenArticles []string
	reader := bufio.NewReader(file)
	// Vaso, ne zaboravi, nema while
	for {
		// postoji i ReadLine ali je on low-level pa se koristi ovo
		article, err := reader.ReadString('\n')
		if err != nil {
			break
		}
		article = strings.TrimSpace(article)
		seenArticles = append(seenArticles, article)
	}

	return seenArticles
}

// Proverava da li slice sadrzi neki string
func contains(target string, slice []string) bool {
	for _, s := range slice {
		if s == target { // odusevljen sam sto mozes string s1 == s2 - komentar coveka koji nije video nista sem c-a #14323
			return true
		}
	}
	return false
}

func saveSeenArticles(seenArticles []string) {
	file, err := os.Create("seen_articles.txt")
	if err != nil {
		log.Println("Error:", err)
	}
	defer file.Close()

	writer := bufio.NewWriter(file)
	for _, article := range seenArticles {
		// u sustini ovaj bufio je kao mng fensi elegantna verzija onih low-level c-ovskih bafera
		// punis neki text u bafer i onda ga flush-ujes u fajl, brutalno
		// inace ovo potencijalno vraca gresku (citaj dokumentaciju (ne)), ali nam greska nije bitna
		// jer smo u fazonu 'daj sta das' pa koliko god da sacuva dobro je
		writer.WriteString(article + "\n")
		writer.Flush()
	}
}
