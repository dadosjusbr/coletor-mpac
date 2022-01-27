package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/chromedp/cdproto/browser"
	"github.com/chromedp/chromedp"
)

type crawler struct {
	collectionTimeout time.Duration
	timeBetweenSteps  time.Duration
	year              string
	month             string
	output            string
}

func (c crawler) crawl() ([]string, error) {
	// Chromedp setup.
	log.SetOutput(os.Stderr) // Enviando logs para o stderr para não afetar a execução do coletor.
	alloc, allocCancel := chromedp.NewExecAllocator(
		context.Background(),
		append(chromedp.DefaultExecAllocatorOptions[:],
			chromedp.UserAgent("Mozilla/5.0 (Macintosh; Intel Mac OS X 10_14_5) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/77.0.3830.0 Safari/537.36"),
			chromedp.Flag("headless", true), // mude para false para executar com navegador visível.
			chromedp.NoSandbox,
			chromedp.DisableGPU,
		)...,
	)
	defer allocCancel()

	ctx, cancel := chromedp.NewContext(
		alloc,
		chromedp.WithLogf(log.Printf), // remover comentário para depurar
	)
	defer cancel()

	ctx, cancel = context.WithTimeout(ctx, c.collectionTimeout)
	defer cancel()

	// NOTA IMPORTANTE: os prefixos dos nomes dos arquivos tem que ser igual
	// ao esperado no parser MPAC.

	// Contracheque
	log.Printf("Entrando em Contracheque(%s/%s)...", c.month, c.year)
	if err := c.navegacaoSite(ctx, "contra"); err != nil {
		log.Fatalf("Erro no setup:%v", err)
	}
	log.Printf("Entrado com sucesso!\n")

	log.Printf("Realizando seleção (%s/%s)...", c.month, c.year)
	if err := c.selecionaMesAno(ctx); err != nil {
		log.Fatalf("Erro no setup:%v", err)
	}
	log.Printf("Seleção realizada com sucesso!\n")
	cqFname := c.downloadFilePath("contracheque")
	log.Printf("Fazendo download do contracheque (%s)...", cqFname)

	if err := c.exportaPlanilha(ctx, cqFname); err != nil {
		log.Fatalf("Erro fazendo download do contracheque: %v", err)
	}
	log.Printf("Download realizado com sucesso!\n")

	// Indenizações
	log.Printf("Entrando em indenezações(%s/%s)...", c.month, c.year)
	if err := c.navegacaoSite(ctx, "inde"); err != nil {
		log.Fatalf("Erro no setup:%v", err)
	}
	log.Printf("Entrado com sucesso!\n")

	log.Printf("Realizando seleção (%s/%s)...", c.month, c.year)
	if err := c.selecionaMesAno(ctx); err != nil {
		log.Fatalf("Erro no setup:%v", err)
	}

	log.Printf("Seleção realizada com sucesso!\n")
	iFname := c.downloadFilePath("verbas-indenizatorias")
	log.Printf("Fazendo download das indenizações (%s)...", iFname)

	if err := c.exportaPlanilha(ctx, iFname); err != nil {
		log.Fatalf("Erro fazendo download dos indenizações: %v", err)
	}
	log.Printf("Download realizado com sucesso!\n")

	// Retorna caminhos completos dos arquivos baixados.
	return []string{cqFname, iFname}, nil
}

func (c crawler) downloadFilePath(prefix string) string {
	return filepath.Join(c.output, fmt.Sprintf("membros-ativos-%s-%s-%s.xlsx", prefix, c.month, c.year))
}

// Navega para as planilhas
func (c crawler) navegacaoSite(ctx context.Context, tipo string) error {
	var baseURL string
	if tipo == "contra" {
		baseURL = "http://transparencia.mpac.mp.br/categoria_arquivos/112"
	} else {
		baseURL = "http://transparencia.mpac.mp.br/categoria_arquivos/119"
	}

	return chromedp.Run(ctx,
		chromedp.Navigate(baseURL),
		chromedp.Sleep(c.timeBetweenSteps),

		// Altera o diretório de download
		browser.SetDownloadBehavior(browser.SetDownloadBehaviorBehaviorAllowAndName).
			WithDownloadPath(c.output).
			WithEventsEnabled(true),
	)
}

func (c crawler) selecionaMesAno(ctx context.Context) error {
	// Coverte para int para tirar o 0 da frente dos numeros
	month, _ := strconv.Atoi(c.month)
	// Coverte para string por conta do chromedp aceitar apenas string como argumento
	m := strconv.Itoa(month)
	
	return chromedp.Run(ctx,
		// Seleciona o ano
		chromedp.SetValue(`//select[@id="ano"]`, c.year, chromedp.BySearch),
		chromedp.Sleep(c.timeBetweenSteps),

		// Seleciona o mes
		chromedp.SetValue(`//select[@id="numMes"]`, m, chromedp.BySearch),
		chromedp.Sleep(c.timeBetweenSteps),
	)	
}

// exportaPlanilha clica no botão correto para exportar para excel, espera um tempo para download renomeia o arquivo.
func (c crawler) exportaPlanilha(ctx context.Context, fName string) error {
	pathPlan := "//*[@id='pesquisaReceita']/input"

	chromedp.Run(ctx,
		// Clica no botão de download
		chromedp.Click(pathPlan, chromedp.BySearch, chromedp.NodeVisible),
		chromedp.Sleep(c.timeBetweenSteps),
	)

	if err := nomeiaDownload(c.output, fName); err != nil {
		return fmt.Errorf("erro renomeando arquivo (%s): %v", fName, err)
	}
	if _, err := os.Stat(fName); os.IsNotExist(err) {
		return fmt.Errorf("download do arquivo de %s não realizado", fName)
	}
	return nil
}

// nomeiaDownload dá um nome ao último arquivo modificado dentro do diretório
// passado como parâmetro nomeiaDownload dá pega um arquivo
func nomeiaDownload(output, fName string) error {
	// Identifica qual foi o ultimo arquivo
	files, err := os.ReadDir(output)
	if err != nil {
		return fmt.Errorf("erro lendo diretório %s: %v", output, err)
	}
	var newestFPath string
	var newestTime int64 = 0
	for _, f := range files {
		fPath := filepath.Join(output, f.Name())
		fi, err := os.Stat(fPath)
		if err != nil {
			return fmt.Errorf("erro obtendo informações sobre arquivo %s: %v", fPath, err)
		}
		currTime := fi.ModTime().Unix()
		if currTime > newestTime {
			newestTime = currTime
			newestFPath = fPath
		}
	}
	// Renomeia o ultimo arquivo modificado.
	if err := os.Rename(newestFPath, fName); err != nil {
		return fmt.Errorf("erro renomeando último arquivo modificado (%s)->(%s): %v", newestFPath, fName, err)
	}
	return nil
}
