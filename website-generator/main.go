package main

import (
	"bytes"
	"flag"
	"fmt"
	"html/template"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/renderer/html"
)

type Module struct {
	Number      int
	Title       string
	Description string
	Filename    string
	Content     template.HTML
	PrevLink    string
	NextLink    string
	Lang        string
	LangLinks   []LangLink
	CSSPath     string
	HomePath    string
	UISiteName  string
	UIHome      string
	UIPrevious  string
	UINext      string
	UIModule    string
	UITrack     string
	UICreatedBy string
	Tracks      []Track
}

type IndexData struct {
	Modules   []Module
	Lang      string
	LangLinks []LangLink
	CSSPath   string
	HomePath  string
}

// LangLink is one language option in the navbar switcher.
type LangLink struct {
	Name     string
	URL      string
	Selected bool
}

type moduleMeta struct {
	Filename    string
	Title       string
	Description string
}

// Link is an entry in the "Resources" section of the index page.
type Link struct {
	Name string
	URL  string
}

// Track is one code track through the workshop. Every module is written once;
// the parts that differ per language live in `<!-- track:ID -->` blocks in the
// markdown, and the reader switches between them on the page.
//
// Adding a track means adding an entry here and writing the matching blocks in
// the markdown. Nothing else in the generator needs to change.
type Track struct {
	ID   string // marker id used in the markdown, e.g. "js"
	Name string // display name in the switcher and the block label
}

// Video is an embedded talk on the index page. Leave the list empty and the
// whole section disappears.
type Video struct {
	Title       string
	EmbedURL    string
	Description string
}

type LangConfig struct {
	Code         string
	FileSuffix   string // ".md" for English, ".es.md" for Spanish
	OutputPrefix string // "" for English, "es" for Spanish
	DisplayName  string // shown in the language switcher (e.g. "English", "Español")
	Metadata     []moduleMeta
	UIStrings    UIStrings
}

type UIStrings struct {
	Home                string
	Previous            string
	Next                string
	Module              string
	Track               string
	TrackHint           string
	HeroTitle           string
	HeroLead            string
	HeroNote            string
	Prerequisites       string
	PrereqItems         []string
	Overview            string
	OverviewText        string
	GettingStarted      string
	GettingStartedItems []string
	Tips                string
	TipItems            []string
	Resources           string
	ResourceLinks       []Link
	VideoReferences     string
	VideoRefsIntro      string
	Videos              []Video
	Completion          string
	CompletionIntro     string
	CompletionItems     []string
	CompletionCongrats  string
	CompletionEnables   []string
	Contributing        string
	ContributingText    string
	CTAButton           string
	FooterTitle         string
	FooterCreatedBy     string
}

const (
	siteName = "Instrumenting your application with OpenTelemetry"
	repoURL  = "https://github.com/jespino/observability-workshop"
)

// tracks are the code tracks offered by the workshop. The first one is the
// default a reader sees before they pick.
//
// To add a track: add an entry here, then add a `<!-- track:ID -->` block
// wherever a module shows code, in every language. `make website` fails loudly
// if a module references a track id that is not listed here.
var tracks = []Track{
	{ID: "js", Name: "Node.js / Express"},
	{ID: "python", Name: "Python / Flask"},
}

func trackByID(id string) (Track, bool) {
	for _, t := range tracks {
		if t.ID == id {
			return t, true
		}
	}
	return Track{}, false
}

var englishConfig = LangConfig{
	Code:         "en",
	FileSuffix:   ".md",
	OutputPrefix: "",
	DisplayName:  "English",
	Metadata: []moduleMeta{
		{"00-setup-and-sanity-check", "Setup and Sanity Check", "Start the whole stack, your API included, with one Docker Compose command, and check with the doctor script that everything is up and running."},
		{"01-the-application", "The Application You'll Instrument", "Tour the todo-api you'll be editing all day — its endpoints, its dependencies, and the handful of functions that matter."},
		{"02-what-observability-is", "The Three Signals — Metrics, Logs and Traces", "What metrics, logs and traces are, with an example of each taken from the application you'll instrument."},
		{"03-auto-instrumentation", "The Easy Path: Zero-Code Auto-Instrumentation", "Get all three signals flowing without touching application code, then find the ceiling of what that buys you."},
		{"04-the-collector", "The Collector, and Where the Data Actually Goes", "Read and edit the OpenTelemetry Collector config that fans a single OTLP endpoint out to three backends."},
		{"05-manual-metrics", "Manual Metrics", "Pick the right instrument, emit business metrics, shape them with views, and meet cardinality before it bites."},
		{"06-manual-logs", "Manual Logs", "Emit structured logs that carry trace context, and query them with LogsQL."},
		{"07-manual-traces", "Manual Traces", "Create custom spans, record errors properly, and propagate context across services and languages."},
		{"08-correlation", "Correlation: Making the Three Signals One Thing", "Derive metrics from spans and wire up the jumps between metrics, logs and traces in Grafana."},
		{"09-capstone-debugging-challenge", "Capstone: The Debugging Challenge", "Diagnose an unfamiliar, production-ish failure using only the telemetry you built."},
		{"10-production-realities", "Production Realities and Wrap-Up", "Cost, sampling, retention, PII and rollout strategy — what changes when this leaves your laptop."},
	},
	UIStrings: UIStrings{
		Home:      "Home",
		Previous:  "Previous",
		Next:      "Next",
		Module:    "Module",
		Track:     "Track",
		TrackHint: `Every module is written once for both tracks. Use the <strong>track switcher</strong> in the navigation bar of any module page to flip the code examples between Node.js and Python — your choice is remembered as you move through the workshop.`,
		HeroTitle: siteName,
		HeroLead:  "Welcome to a hands-on workshop where you'll instrument a multi-service application from scratch with OpenTelemetry, and ship metrics, logs and traces into the VictoriaMetrics stack. Roughly 25% talk, 75% keyboard.",
		HeroNote:  "<strong>Two parallel tracks: Node.js/Express and Python/Flask.</strong> Same system, same backend, same exercises — only the SDK surface differs.",

		Prerequisites: "Prerequisites",
		PrereqItems: []string{
			"Comfortable with HTTP APIs and either JavaScript or Python",
			"Docker and Docker Compose installed and working",
			"Git installed on your system",
			"<strong>No prior OpenTelemetry, MetricsQL or LogsQL knowledge needed</strong>",
			"At least 8GB of free RAM and ~5GB of disk space",
		},
		Overview:       "Workshop Overview",
		OverviewText:   "This workshop consists of %d modules that take you from a service emitting nothing at all to one you can debug from telemetry alone. You'll instrument a multi-service system, run an OpenTelemetry Collector as the single ingress point, and query the results in VictoriaMetrics, VictoriaLogs, VictoriaTraces and Grafana:",
		GettingStarted: "Getting Started",
		GettingStartedItems: []string{
			`Start with <a href="%s00-setup-and-sanity-check.html">Module 0</a> to get your environment ready`,
			"Work through the modules in order — each one builds on the last",
			"Pick the track you want to type in, JavaScript or Python. Both end up looking at identical data.",
		},
		Tips: "Tips for Success",
		TipItems: []string{
			"Paste the exact queries from the handout — \"just explore the data\" comes later",
			"When something looks wrong, read the Collector logs before anything else",
			"You can look at the solutions in the repository",
		},
		Resources: "Resources",
		ResourceLinks: []Link{
			{"OpenTelemetry Documentation", "https://opentelemetry.io/docs/"},
			{"OpenTelemetry Semantic Conventions", "https://opentelemetry.io/docs/specs/semconv/"},
			{"OpenTelemetry Collector", "https://opentelemetry.io/docs/collector/"},
			{"VictoriaMetrics MetricsQL", "https://docs.victoriametrics.com/metricsql/"},
			{"VictoriaLogs LogsQL", "https://docs.victoriametrics.com/victorialogs/logsql/"},
			{"VictoriaTraces", "https://docs.victoriametrics.com/victoriatraces/"},
		},
		VideoReferences: "Video References",
		VideoRefsIntro:  "Talks that go deeper on the ideas behind these modules:",
		Videos:          []Video{},
		Completion:      "Workshop Completion",
		CompletionIntro: "Upon completing all modules, you'll have:",
		CompletionItems: []string{
			"<strong>Instrumented a service</strong> with metrics, logs and traces, starting from nothing",
			"<strong>Run an OpenTelemetry Collector</strong> as a single ingress point, fanning out to three backends",
			"<strong>Chosen the right instruments</strong> and reasoned about cardinality before it hurt",
			"<strong>Emitted structured logs</strong> that carry trace context, and queried them with LogsQL",
			"<strong>Written custom spans</strong> and propagated context across service and language boundaries",
			"<strong>Correlated all three signals</strong> in Grafana and in the Victoria UIs",
			"<strong>Debugged an unfamiliar failure</strong> using only telemetry",
		},
		CompletionCongrats: "<strong>Congratulations!</strong> You'll have gained the confidence to instrument your own services deliberately rather than by accident. This knowledge enables you to:",
		CompletionEnables: []string{
			"Add telemetry that answers real questions to services you already own",
			"Design dashboards and alerts around questions, not around whatever was easy to emit",
			"Reason about the cost and cardinality of everything you collect",
		},
		Contributing:     "Contributing",
		ContributingText: `Found an issue, have an improvement idea or want to add more modules? Please <a href="` + repoURL + `/issues">open an issue</a> or submit a pull request!`,
		CTAButton:        "Start with Module 0 →",
		FooterTitle:      siteName,
		FooterCreatedBy:  "Created by <strong>Jesús Espino</strong>",
	},
}

var spanishConfig = LangConfig{
	Code:         "es",
	FileSuffix:   ".es.md",
	OutputPrefix: "es",
	DisplayName:  "Español",
	Metadata: []moduleMeta{
		{"00-setup-and-sanity-check", "Preparación y Comprobación del Entorno", "Arranca todo el stack, tu API incluida, con un solo comando de Docker Compose, y comprueba con el script doctor que todo está funcionando."},
		{"01-the-application", "La Aplicación que Vas a Instrumentar", "Recorre la todo-api que vas a editar durante todo el día — sus endpoints, sus dependencias y el puñado de funciones que importan."},
		{"02-what-observability-is", "Las Tres Señales — Métricas, Logs y Trazas", "Qué son las métricas, los logs y las trazas, con un ejemplo de cada una sacado de la aplicación que vas a instrumentar."},
		{"03-auto-instrumentation", "El Camino Fácil: Auto-instrumentación Sin Código", "Haz que fluyan las tres señales sin tocar el código de la aplicación, y descubre hasta dónde llega ese atajo."},
		{"04-the-collector", "El Collector, y Dónde Acaban los Datos", "Lee y modifica la configuración del OpenTelemetry Collector que reparte un único endpoint OTLP entre tres backends."},
		{"05-manual-metrics", "Métricas Manuales", "Elige el instrumento correcto, emite métricas de negocio, moldéalas con views y enfréntate a la cardinalidad antes de que te muerda."},
		{"06-manual-logs", "Logs Manuales", "Emite logs estructurados que arrastren el contexto de traza, y consúltalos con LogsQL."},
		{"07-manual-traces", "Trazas Manuales", "Crea spans propios, registra errores como toca y propaga el contexto entre servicios y lenguajes."},
		{"08-correlation", "Correlación: Convertir las Tres Señales en Una", "Deriva métricas a partir de spans y conecta los saltos entre métricas, logs y trazas en Grafana."},
		{"09-capstone-debugging-challenge", "Proyecto Final: El Reto de Depuración", "Diagnostica un fallo desconocido, de los de producción, usando solo la telemetría que has construido."},
		{"10-production-realities", "Realidades de Producción y Cierre", "Coste, sampling, retención, PII y estrategia de despliegue: qué cambia cuando esto sale de tu portátil."},
	},
	UIStrings: UIStrings{
		Home:      "Inicio",
		Previous:  "Anterior",
		Next:      "Siguiente",
		Module:    "Módulo",
		Track:     "Track",
		TrackHint: `Cada módulo está escrito una sola vez para los dos tracks. Usa el <strong>selector de track</strong> de la barra de navegación de cualquier módulo para cambiar los ejemplos de código entre Node.js y Python — tu elección se recuerda mientras avanzas por el taller.`,
		HeroTitle: "Instrumentando tu aplicación con OpenTelemetry",
		HeroLead:  "¡Bienvenido a un taller práctico donde instrumentarás desde cero una aplicación de varios servicios con OpenTelemetry, y enviarás métricas, logs y trazas al stack de VictoriaMetrics! Aproximadamente 25% charla, 75% teclado.",
		HeroNote:  "<strong>Dos tracks en paralelo: Node.js/Express y Python/Flask.</strong> El mismo sistema, el mismo backend y los mismos ejercicios — solo cambia la superficie del SDK.",

		Prerequisites: "Prerrequisitos",
		PrereqItems: []string{
			"Soltura con APIs HTTP y con JavaScript o Python",
			"Docker y Docker Compose instalados y funcionando",
			"Git instalado en tu sistema",
			"<strong>No hace falta saber nada de OpenTelemetry, MetricsQL ni LogsQL</strong>",
			"Al menos 8GB de RAM libre y unos 5GB de espacio en disco",
		},
		Overview:       "Descripción General del Taller",
		OverviewText:   "Este taller consta de %d módulos que te llevan desde un servicio que no emite absolutamente nada hasta uno que puedes depurar solo con telemetría. Instrumentarás un sistema de varios servicios, ejecutarás un OpenTelemetry Collector como único punto de entrada y consultarás los resultados en VictoriaMetrics, VictoriaLogs, VictoriaTraces y Grafana:",
		GettingStarted: "Cómo Empezar",
		GettingStartedItems: []string{
			`Comienza con el <a href="%s00-setup-and-sanity-check.html">Módulo 0</a> para dejar tu entorno preparado`,
			"Haz los módulos en orden — cada uno se apoya en el anterior",
			"Elige el track en el que quieras escribir, JavaScript o Python. Los dos acaban mirando exactamente los mismos datos.",
		},
		Tips: "Consejos para el Éxito",
		TipItems: []string{
			"Pega las consultas exactas del guion — lo de \"explora los datos\" viene después",
			"Cuando algo pinte mal, lee los logs del Collector antes que nada",
			"Puedes mirar las soluciones en el repositorio",
		},
		Resources: "Recursos",
		ResourceLinks: []Link{
			{"Documentación de OpenTelemetry", "https://opentelemetry.io/docs/"},
			{"Convenciones Semánticas de OpenTelemetry", "https://opentelemetry.io/docs/specs/semconv/"},
			{"OpenTelemetry Collector", "https://opentelemetry.io/docs/collector/"},
			{"MetricsQL de VictoriaMetrics", "https://docs.victoriametrics.com/metricsql/"},
			{"LogsQL de VictoriaLogs", "https://docs.victoriametrics.com/victorialogs/logsql/"},
			{"VictoriaTraces", "https://docs.victoriametrics.com/victoriatraces/"},
		},
		VideoReferences: "Referencias en Video",
		VideoRefsIntro:  "Charlas que profundizan en las ideas detrás de estos módulos:",
		Videos:          []Video{},
		Completion:      "Completando el Taller",
		CompletionIntro: "Al completar todos los módulos, habrás:",
		CompletionItems: []string{
			"<strong>Instrumentado un servicio</strong> con métricas, logs y trazas, partiendo de cero",
			"<strong>Ejecutado un OpenTelemetry Collector</strong> como punto de entrada único, repartiendo hacia tres backends",
			"<strong>Elegido los instrumentos correctos</strong> y razonado sobre cardinalidad antes de que doliera",
			"<strong>Emitido logs estructurados</strong> que arrastran el contexto de traza, y consultado con LogsQL",
			"<strong>Escrito spans propios</strong> y propagado el contexto entre servicios y lenguajes",
			"<strong>Correlacionado las tres señales</strong> en Grafana y en las UIs de Victoria",
			"<strong>Depurado un fallo desconocido</strong> usando solo telemetría",
		},
		CompletionCongrats: "<strong>¡Felicidades!</strong> Habrás ganado la confianza para instrumentar tus propios servicios de forma deliberada y no por accidente. Este conocimiento te permite:",
		CompletionEnables: []string{
			"Añadir a tus servicios telemetría que responda preguntas reales",
			"Diseñar dashboards y alertas alrededor de preguntas, no de lo que resultó fácil de emitir",
			"Razonar sobre el coste y la cardinalidad de todo lo que recoges",
		},
		Contributing:     "Contribuir",
		ContributingText: `¿Encontraste un problema, tienes una idea de mejora o quieres añadir más módulos? ¡Por favor <a href="` + repoURL + `/issues">abre un issue</a> o envía un pull request!`,
		CTAButton:        "Comenzar con el Módulo 0 →",
		FooterTitle:      "Instrumentando tu aplicación con OpenTelemetry",
		FooterCreatedBy:  "Creado por <strong>Jesús Espino</strong>",
	},
}

var languages = []LangConfig{englishConfig, spanishConfig}

// computeLangURL builds a relative URL from fromLang's page directory to toLang's page.
// page is the HTML filename (e.g. "index.html" or "00-setup-and-sanity-check.html").
func computeLangURL(fromLang, toLang LangConfig, page string) string {
	fromSub := fromLang.OutputPrefix != ""
	toSub := toLang.OutputPrefix != ""

	switch {
	case !fromSub && !toSub:
		return page
	case !fromSub && toSub:
		return toLang.OutputPrefix + "/" + page
	case fromSub && !toSub:
		return "../" + page
	default:
		return "../" + toLang.OutputPrefix + "/" + page
	}
}

// langSwitcherLinks returns all language options for the navbar dropdown.
// The current language is included as the selected option (empty URL so re-select is a no-op).
func langSwitcherLinks(current LangConfig, page string) []LangLink {
	links := make([]LangLink, 0, len(languages))
	for _, lang := range languages {
		if lang.Code == current.Code {
			links = append(links, LangLink{
				Name:     lang.DisplayName,
				URL:      "",
				Selected: true,
			})
			continue
		}
		links = append(links, LangLink{
			Name:     lang.DisplayName,
			URL:      computeLangURL(current, lang, page),
			Selected: false,
		})
	}
	return links
}

func main() {
	modulesDir := flag.String("modules", "../modules", "Path to modules directory")
	outputDir := flag.String("output", "../website", "Path to output directory")
	serve := flag.Bool("serve", false, "Start dev server with live reload")
	port := flag.Int("port", 8000, "Dev server port (used with -serve)")
	flag.Parse()

	if *serve {
		srv := newDevServer(*modulesDir, *outputDir, *port)
		if err := srv.run(); err != nil {
			fmt.Fprintf(os.Stderr, "Server error: %v\n", err)
			os.Exit(1)
		}
		return
	}

	// Create output directory if it doesn't exist
	if err := os.MkdirAll(*outputDir, 0o755); err != nil {
		fmt.Fprintf(os.Stderr, "Error creating output directory: %v\n", err)
		os.Exit(1)
	}

	totalPages := 0
	for _, lang := range languages {
		// Determine output directory for this language
		langOutputDir := *outputDir
		if lang.OutputPrefix != "" {
			langOutputDir = filepath.Join(*outputDir, lang.OutputPrefix)
			if err := os.MkdirAll(langOutputDir, 0o755); err != nil {
				fmt.Fprintf(os.Stderr, "Error creating output directory for %s: %v\n", lang.Code, err)
				os.Exit(1)
			}
		}

		// Determine CSS path relative to output dir
		cssPath := "style.css"
		if lang.OutputPrefix != "" {
			cssPath = "../style.css"
		}

		homePath := ""

		// Generate module pages
		modules := make([]Module, 0, len(lang.Metadata))
		for i, meta := range lang.Metadata {
			module, err := generateModulePage(*modulesDir, langOutputDir, lang, meta, i, cssPath, homePath)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error generating module %s (%s): %v\n", meta.Filename, lang.Code, err)
				os.Exit(1)
			}
			modules = append(modules, module)
		}

		// Generate index page
		if err := generateIndexPage(langOutputDir, lang, modules, cssPath, homePath); err != nil {
			fmt.Fprintf(os.Stderr, "Error generating index page (%s): %v\n", lang.Code, err)
			os.Exit(1)
		}

		totalPages += len(modules) + 1
	}

	// Copy CSS file (only at root level, shared by all languages)
	if err := copyCSSFile(*outputDir); err != nil {
		fmt.Fprintf(os.Stderr, "Error copying CSS file: %v\n", err)
		os.Exit(1)
	}

	// Copy images (only at root level, shared by all languages)
	if err := copyImages(*modulesDir, *outputDir); err != nil {
		fmt.Fprintf(os.Stderr, "Error copying images: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("✅ Website generated successfully!")
	fmt.Printf("📁 Output directory: %s\n", *outputDir)
	fmt.Printf("📄 Generated %d pages total (including all languages)\n", totalPages)
}

func generateModulePage(modulesDir, outputDir string, lang LangConfig, meta moduleMeta, index int, cssPath, homePath string) (Module, error) {
	// Read markdown file
	mdFilename := meta.Filename + lang.FileSuffix
	mdPath := filepath.Join(modulesDir, mdFilename)
	content, err := os.ReadFile(mdPath)
	if err != nil {
		return Module{}, fmt.Errorf("reading markdown file: %w", err)
	}

	// Convert markdown to HTML
	htmlContent, err := markdownToHTML(content)
	if err != nil {
		return Module{}, fmt.Errorf("rendering %s: %w", mdFilename, err)
	}

	// Handouts reference images as images/NAME, relative to modules/. The
	// images are copied once to the site root, so pages in a language
	// subdirectory need to reach one level up — the same as the stylesheet.
	if prefix := strings.TrimSuffix(cssPath, "style.css"); prefix != "" {
		htmlContent = strings.ReplaceAll(htmlContent, `src="images/`, `src="`+prefix+`images/`)
	}

	// Generate HTML filename
	htmlFilename := meta.Filename + ".html"

	// Determine prev/next links
	prevLink := homePath + "index.html"
	if index > 0 {
		prevLink = lang.Metadata[index-1].Filename + ".html"
	}

	nextLink := ""
	if index < len(lang.Metadata)-1 {
		nextLink = lang.Metadata[index+1].Filename + ".html"
	}

	// Footer "Created by" label without the author HTML
	createdBy := "Created by"
	if lang.Code == "es" {
		createdBy = "Creado por"
	}

	module := Module{
		Number:      index,
		Title:       meta.Title,
		Description: meta.Description,
		Filename:    htmlFilename,
		Content:     template.HTML(htmlContent),
		PrevLink:    prevLink,
		NextLink:    nextLink,
		Lang:        lang.Code,
		LangLinks:   langSwitcherLinks(lang, htmlFilename),
		CSSPath:     cssPath,
		HomePath:    homePath,
		UISiteName:  lang.UIStrings.HeroTitle,
		UIHome:      lang.UIStrings.Home,
		UIPrevious:  lang.UIStrings.Previous,
		UINext:      lang.UIStrings.Next,
		UIModule:    lang.UIStrings.Module,
		UITrack:     lang.UIStrings.Track,
		UICreatedBy: createdBy,
		Tracks:      tracks,
	}

	// Generate HTML page
	tmpl, err := template.New("module").Funcs(template.FuncMap{
		"add": func(a, b int) int {
			return a + b
		},
	}).Parse(moduleTemplate)
	if err != nil {
		return Module{}, fmt.Errorf("parsing template: %w", err)
	}

	outputPath := filepath.Join(outputDir, htmlFilename)
	f, err := os.Create(outputPath)
	if err != nil {
		return Module{}, fmt.Errorf("creating output file: %w", err)
	}
	defer f.Close()

	if err := tmpl.Execute(f, module); err != nil {
		return Module{}, fmt.Errorf("executing template: %w", err)
	}

	fmt.Printf("✓ Generated %s [%s]\n", htmlFilename, lang.Code)
	return module, nil
}

func generateIndexPage(outputDir string, lang LangConfig, modules []Module, cssPath, homePath string) error {
	tmpl, err := template.New("index").Funcs(template.FuncMap{
		"safeHTML": func(s string) template.HTML {
			return template.HTML(s)
		},
	}).Parse(indexTemplate)
	if err != nil {
		return fmt.Errorf("parsing template: %w", err)
	}

	outputPath := filepath.Join(outputDir, "index.html")
	f, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("creating output file: %w", err)
	}
	defer f.Close()

	// Format overview text with module count
	ui := lang.UIStrings
	ui.OverviewText = fmt.Sprintf(ui.OverviewText, len(modules))

	// Format getting started items with link prefix
	formattedGSItems := make([]string, len(ui.GettingStartedItems))
	for i, item := range ui.GettingStartedItems {
		if strings.Contains(item, "%s") {
			formattedGSItems[i] = fmt.Sprintf(item, "")
		} else {
			formattedGSItems[i] = item
		}
	}
	ui.GettingStartedItems = formattedGSItems

	data := struct {
		IndexData
		UI UIStrings
	}{
		IndexData: IndexData{
			Modules:   modules,
			Lang:      lang.Code,
			LangLinks: langSwitcherLinks(lang, "index.html"),
			CSSPath:   cssPath,
			HomePath:  homePath,
		},
		UI: ui,
	}
	if err := tmpl.Execute(f, data); err != nil {
		return fmt.Errorf("executing template: %w", err)
	}

	fmt.Printf("✓ Generated index.html [%s]\n", lang.Code)
	return nil
}

// copyImages copies modules/images/ to the site's images/ directory. A missing
// modules/images/ is not an error: not every build has images.
func copyImages(modulesDir, outputDir string) error {
	srcDir := filepath.Join(modulesDir, "images")
	entries, err := os.ReadDir(srcDir)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("reading %s: %w", srcDir, err)
	}

	dstDir := filepath.Join(outputDir, "images")
	if err := os.MkdirAll(dstDir, 0o755); err != nil {
		return fmt.Errorf("creating %s: %w", dstDir, err)
	}
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		data, err := os.ReadFile(filepath.Join(srcDir, e.Name()))
		if err != nil {
			return fmt.Errorf("reading image: %w", err)
		}
		if err := os.WriteFile(filepath.Join(dstDir, e.Name()), data, 0o644); err != nil {
			return fmt.Errorf("writing image: %w", err)
		}
	}
	return nil
}

func copyCSSFile(outputDir string) error {
	cssContent := cssTemplate
	outputPath := filepath.Join(outputDir, "style.css")

	if err := os.WriteFile(outputPath, []byte(cssContent), 0o644); err != nil {
		return fmt.Errorf("writing CSS file: %w", err)
	}

	fmt.Printf("✓ Generated style.css\n")
	return nil
}

// md is the Markdown converter: CommonMark plus the GitHub extensions the
// handouts use (tables above all). It replaced blackfriday, which silently
// rendered a table nested in a list item, or a code fence inside a
// blockquote, as plain text.
var md = goldmark.New(
	goldmark.WithExtensions(extension.GFM, extension.Typographer),
	// Raw HTML must pass through: the track markers are HTML comments.
	goldmark.WithRendererOptions(html.WithUnsafe()),
)

func markdownToHTML(markdown []byte) (string, error) {
	// Goldmark passes HTML comments through verbatim and still parses the
	// markdown between them, which is what makes the track markers work.
	var buf bytes.Buffer
	if err := md.Convert(markdown, &buf); err != nil {
		return "", fmt.Errorf("converting markdown: %w", err)
	}

	// Post-process to fix relative links
	htmlStr := buf.String()
	htmlStr = fixRelativeLinks(htmlStr)

	// Turn the track markers into switchable blocks
	htmlStr, err := applyTrackBlocks(htmlStr)
	if err != nil {
		return "", err
	}

	return htmlStr, nil
}

var (
	trackOpen  = regexp.MustCompile(`<!--\s*track:([a-zA-Z0-9_-]+)\s*-->`)
	trackClose = regexp.MustCompile(`<!--\s*/track\s*-->`)
)

// applyTrackBlocks rewrites the `<!-- track:ID -->` / `<!-- /track -->` markers
// into divs the page-level switcher can show and hide. Blocks are labelled so
// the page still reads correctly with JavaScript disabled, and so it prints.
//
// Unknown track ids and unbalanced markers are errors: a typo in a handout
// should fail the build, not silently hide an exercise from half the room.
func applyTrackBlocks(html string) (string, error) {
	var (
		out   strings.Builder
		depth int
		rest  = html
	)

	for {
		opens := trackOpen.FindStringSubmatchIndex(rest)
		closes := trackClose.FindStringIndex(rest)

		switch {
		case opens == nil && closes == nil:
			out.WriteString(rest)
			if depth != 0 {
				return "", fmt.Errorf("unclosed <!-- track:... --> block")
			}
			return out.String(), nil

		case closes == nil || (opens != nil && opens[0] < closes[0]):
			if depth != 0 {
				return "", fmt.Errorf("nested <!-- track:... --> block")
			}
			id := rest[opens[2]:opens[3]]
			track, ok := trackByID(id)
			if !ok {
				return "", fmt.Errorf("unknown track %q (known tracks: %s)", id, trackIDs())
			}
			out.WriteString(rest[:opens[0]])
			fmt.Fprintf(&out, `<div class="track-block" data-track="%s"><p class="track-label">%s</p>`,
				template.HTMLEscapeString(track.ID), template.HTMLEscapeString(track.Name))
			rest = rest[opens[1]:]
			depth++

		default:
			if depth == 0 {
				return "", fmt.Errorf("<!-- /track --> without a matching <!-- track:... -->")
			}
			out.WriteString(rest[:closes[0]])
			out.WriteString("</div>")
			rest = rest[closes[1]:]
			depth--
		}
	}
}

func trackIDs() string {
	ids := make([]string, 0, len(tracks))
	for _, t := range tracks {
		ids = append(ids, t.ID)
	}
	return strings.Join(ids, ", ")
}

// docsOnGitHub matches links to repository documents that have no page on the
// generated site (the facilitator design doc, for instance). They are rewritten to
// point at GitHub rather than 404.
var docsOnGitHub = regexp.MustCompile(`href="\.\./(workshop\.md|LICENSE)"`)

func fixRelativeLinks(html string) string {
	// A README link in any language is the site's index page
	re := regexp.MustCompile(`href="\.\./README(?:\.es)?\.md"`)
	html = re.ReplaceAllString(html, `href="index.html"`)

	// Links into the modules directory become sibling HTML pages
	re = regexp.MustCompile(`href="\.\./modules/([^"]+?)(?:\.es)?\.md"`)
	html = re.ReplaceAllString(html, `href="$1.html"`)

	// Repository documents that are not part of the site
	html = docsOnGitHub.ReplaceAllString(html, fmt.Sprintf(`href="%s/blob/main/$1"`, repoURL))

	// Fix links that are already in the format XX-name.md or XX-name.es.md
	re = regexp.MustCompile(`href="(?:\./)?([0-9]{2}-[^"]+?)(?:\.es)?\.md"`)
	html = re.ReplaceAllString(html, `href="$1.html"`)

	return html
}

const moduleTemplate = `<!DOCTYPE html>
<html lang="{{.Lang}}">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>{{.UIModule}} {{.Number}}: {{.Title}} - Observability Workshop</title>
    <link rel="preconnect" href="https://fonts.googleapis.com">
    <link rel="preconnect" href="https://fonts.gstatic.com" crossorigin>
    <link rel="stylesheet" href="https://fonts.googleapis.com/css2?family=Lato:wght@400;700;900&family=Ubuntu+Mono:wght@400;700&display=swap">
    <link rel="stylesheet" href="{{.CSSPath}}">
    <link rel="stylesheet" href="https://cdnjs.cloudflare.com/ajax/libs/highlight.js/11.9.0/styles/atom-one-dark.min.css">
    <link rel="stylesheet" href="https://cdnjs.cloudflare.com/ajax/libs/font-awesome/6.5.1/css/all.min.css">
    <script src="https://cdnjs.cloudflare.com/ajax/libs/highlight.js/11.9.0/highlight.min.js"></script>
    <script src="https://cdnjs.cloudflare.com/ajax/libs/highlight.js/11.9.0/languages/dockerfile.min.js"></script>
    <script>
        document.addEventListener('DOMContentLoaded', function() {
            hljs.highlightAll();

            // Add copy buttons to all code blocks
            document.querySelectorAll('pre').forEach(function(pre) {
                const button = document.createElement('button');
                button.className = 'copy-button';
                button.innerHTML = '<i class="far fa-copy"></i>';
                button.title = 'Copy to clipboard';

                button.addEventListener('click', function() {
                    const code = pre.querySelector('code');
                    const text = code.textContent;

                    navigator.clipboard.writeText(text).then(function() {
                        button.innerHTML = '<i class="fas fa-check"></i>';
                        button.classList.add('copied');
                        setTimeout(function() {
                            button.innerHTML = '<i class="far fa-copy"></i>';
                            button.classList.remove('copied');
                        }, 2000);
                    }).catch(function(err) {
                        console.error('Failed to copy:', err);
                    });
                });

                pre.appendChild(button);
            });
        });

        // Track switcher. The choice is remembered per browser and applied
        // before paint on every page, so switching once carries the reader
        // through the whole workshop.
        (function() {
            var KEY = 'workshop-track';
            var TRACKS = [{{range $i, $t := .Tracks}}{{if $i}}, {{end}}'{{$t.ID}}'{{end}}];

            function stored() {
                try {
                    var value = window.localStorage.getItem(KEY);
                    return TRACKS.indexOf(value) >= 0 ? value : null;
                } catch (e) {
                    return null;
                }
            }

            function remember(track) {
                try { window.localStorage.setItem(KEY, track); } catch (e) { /* private mode */ }
            }

            // Hide the other tracks with a stylesheet rather than by walking
            // the DOM: this runs in <head>, before the blocks exist, so the
            // reader never sees both tracks flash past.
            var style = document.createElement('style');
            document.head.appendChild(style);

            function apply(track) {
                document.documentElement.setAttribute('data-track', track);
                style.textContent =
                    '.track-block{display:none}' +
                    '.track-block[data-track="' + track + '"]{display:block}';
            }

            var current = stored() || TRACKS[0];
            apply(current);

            document.addEventListener('DOMContentLoaded', function() {
                var select = document.getElementById('track-select');
                if (!select) return;
                select.value = current;
                select.addEventListener('change', function() {
                    current = select.value;
                    remember(current);
                    apply(current);
                });
            });
        })();
    </script>
</head>
<body>
    <nav class="navbar">
        <div class="container">
            <a href="{{.HomePath}}index.html" class="nav-home">{{.UISiteName}}</a>
            <div class="nav-links">
                <a href="{{.HomePath}}index.html">{{.UIHome}}</a>
                <label class="track-switch">
                    <i class="fas fa-code" aria-hidden="true"></i>
                    <select id="track-select" aria-label="{{.UITrack}}">
                        {{range .Tracks}}<option value="{{.ID}}">{{.Name}}</option>{{end}}
                    </select>
                </label>
                <label class="lang-switch">
                    <i class="fas fa-globe" aria-hidden="true"></i>
                    <select onchange="if (this.value) location.href = this.value" aria-label="Language">
                        {{range .LangLinks}}<option value="{{.URL}}"{{if .Selected}} selected{{end}}>{{.Name}}</option>{{end}}
                    </select>
                </label>
                <a href="https://github.com/jespino/observability-workshop" target="_blank"><i class="fab fa-github"></i> Repository</a>
            </div>
        </div>
    </nav>

    <div class="container">
        <article class="module-content">
            {{.Content}}
        </article>

        <nav class="module-nav">
            {{if .PrevLink}}
            <a href="{{.PrevLink}}" class="nav-button">{{ if or (eq .PrevLink "index.html") (eq .PrevLink "") }}← {{.UIHome}}{{ else }}← {{.UIPrevious}}{{ end }}</a>
            {{end}}
            {{if .NextLink}}
            <a href="{{.NextLink}}" class="nav-button">{{.UINext}}: {{.UIModule}} {{add .Number 1}} →</a>
            {{end}}
        </nav>
    </div>

    <footer>
        <div class="container">
            <p>{{.UISiteName}}</p>
            <p>{{.UICreatedBy}} <strong>Jesús Espino</strong></p>
            <div class="footer-links">
                <a href="https://github.com/jespino" target="_blank"><i class="fab fa-github"></i> GitHub</a>
                <a href="https://x.com/jespinog" target="_blank"><i class="fab fa-x-twitter"></i> @jespinog</a>
                <a href="https://linkedin.com/in/jesus-espino" target="_blank"><i class="fab fa-linkedin"></i> LinkedIn</a>
            </div>
        </div>
    </footer>
</body>
</html>
`

const indexTemplate = `<!DOCTYPE html>
<html lang="{{.Lang}}">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>{{.UI.HeroTitle}}</title>
    <link rel="preconnect" href="https://fonts.googleapis.com">
    <link rel="preconnect" href="https://fonts.gstatic.com" crossorigin>
    <link rel="stylesheet" href="https://fonts.googleapis.com/css2?family=Lato:wght@400;700;900&family=Ubuntu+Mono:wght@400;700&display=swap">
    <link rel="stylesheet" href="{{.CSSPath}}">
    <link rel="stylesheet" href="https://cdnjs.cloudflare.com/ajax/libs/highlight.js/11.9.0/styles/atom-one-dark.min.css">
    <link rel="stylesheet" href="https://cdnjs.cloudflare.com/ajax/libs/font-awesome/6.5.1/css/all.min.css">
    <script src="https://cdnjs.cloudflare.com/ajax/libs/highlight.js/11.9.0/highlight.min.js"></script>
    <script src="https://cdnjs.cloudflare.com/ajax/libs/highlight.js/11.9.0/languages/dockerfile.min.js"></script>
    <script>
        document.addEventListener('DOMContentLoaded', function() {
            hljs.highlightAll();

            // Add copy buttons to all code blocks
            document.querySelectorAll('pre').forEach(function(pre) {
                const button = document.createElement('button');
                button.className = 'copy-button';
                button.innerHTML = '<i class="far fa-copy"></i>';
                button.title = 'Copy to clipboard';

                button.addEventListener('click', function() {
                    const code = pre.querySelector('code');
                    const text = code.textContent;

                    navigator.clipboard.writeText(text).then(function() {
                        button.innerHTML = '<i class="fas fa-check"></i>';
                        button.classList.add('copied');
                        setTimeout(function() {
                            button.innerHTML = '<i class="far fa-copy"></i>';
                            button.classList.remove('copied');
                        }, 2000);
                    }).catch(function(err) {
                        console.error('Failed to copy:', err);
                    });
                });

                pre.appendChild(button);
            });
        });
    </script>
</head>
<body>
    <nav class="navbar">
        <div class="container">
            <a href="{{.HomePath}}index.html" class="nav-home">{{.UI.HeroTitle}}</a>
            <div class="nav-links">
                <a href="{{.HomePath}}index.html">{{.UI.Home}}</a>
                <label class="lang-switch">
                    <i class="fas fa-globe" aria-hidden="true"></i>
                    <select onchange="if (this.value) location.href = this.value" aria-label="Language">
                        {{range .LangLinks}}<option value="{{.URL}}"{{if .Selected}} selected{{end}}>{{.Name}}</option>{{end}}
                    </select>
                </label>
                <a href="https://github.com/jespino/observability-workshop" target="_blank"><i class="fab fa-github"></i> Repository</a>
            </div>
        </div>
    </nav>

    <div class="container">
        <header class="hero">
            <h1>{{.UI.HeroTitle}}</h1>
            <p class="lead">{{.UI.HeroLead}}</p>
            <p class="hero-note">{{safeHTML .UI.HeroNote}}</p>
        </header>

        <section class="prerequisites">
            <h2>{{.UI.Prerequisites}}</h2>
            <ul>
                {{range .UI.PrereqItems}}<li>{{safeHTML .}}</li>
                {{end}}
            </ul>
        </section>

        <section class="overview">
            <h2>{{.UI.Overview}}</h2>
            <p>{{safeHTML .UI.OverviewText}}</p>
            <p class="track-hint"><i class="fas fa-code" aria-hidden="true"></i> {{safeHTML .UI.TrackHint}}</p>

            <div class="modules-grid">
                {{range .Modules}}
                <a href="{{.Filename}}" class="module-card-link">
                    <div class="module-card">
                        <div class="module-number">{{$.UI.Module}} {{.Number}}</div>
                        <h3>{{.Title}}</h3>
                        <p>{{.Description}}</p>
                    </div>
                </a>
                {{end}}
            </div>
        </section>

        <section class="getting-started">
            <h2>{{.UI.GettingStarted}}</h2>
            <ol>
                {{range .UI.GettingStartedItems}}<li>{{safeHTML .}}</li>
                {{end}}
            </ol>
        </section>

        <section class="tips">
            <h2>{{.UI.Tips}}</h2>
            <ul>
                {{range .UI.TipItems}}<li>{{safeHTML .}}</li>
                {{end}}
            </ul>
        </section>

        <section class="resources">
            <h2>{{.UI.Resources}}</h2>
            <ul>
                {{range .UI.ResourceLinks}}<li><a href="{{.URL}}" target="_blank">{{.Name}}</a></li>
                {{end}}
            </ul>

            {{if .UI.Videos}}
            <h3>{{.UI.VideoReferences}}</h3>
            <p>{{.UI.VideoRefsIntro}}</p>
            <div class="video-grid">
                {{range .UI.Videos}}
                <div class="video-container">
                    <h4>{{.Title}}</h4>
                    <iframe src="{{.EmbedURL}}" frameborder="0" allow="accelerometer; autoplay; clipboard-write; encrypted-media; gyroscope; picture-in-picture" allowfullscreen></iframe>
                    <p>{{.Description}}</p>
                </div>
                {{end}}
            </div>
            {{end}}
        </section>

        <section class="completion">
            <h2>{{.UI.Completion}}</h2>
            <p>{{.UI.CompletionIntro}}</p>
            <ul>
                {{range .UI.CompletionItems}}<li>{{safeHTML .}}</li>
                {{end}}
            </ul>

            <p>{{safeHTML .UI.CompletionCongrats}}</p>
            <ul>
                {{range .UI.CompletionEnables}}<li>{{.}}</li>
                {{end}}
            </ul>
        </section>

        <section class="contributing">
            <h2>{{.UI.Contributing}}</h2>
            <p>{{safeHTML .UI.ContributingText}}</p>
        </section>

        <div class="cta">
            <a href="00-setup-and-sanity-check.html" class="cta-button">{{.UI.CTAButton}}</a>
        </div>
    </div>

    <footer>
        <div class="container">
            <p>{{.UI.FooterTitle}}</p>
            <p>{{safeHTML .UI.FooterCreatedBy}}</p>
            <div class="footer-links">
                <a href="https://github.com/jespino" target="_blank"><i class="fab fa-github"></i> GitHub</a>
                <a href="https://x.com/jespinog" target="_blank"><i class="fab fa-x-twitter"></i> @jespinog</a>
                <a href="https://linkedin.com/in/jesus-espino" target="_blank"><i class="fab fa-linkedin"></i> LinkedIn</a>
            </div>
        </div>
    </footer>
</body>
</html>
`
