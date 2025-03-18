package main

import (
	"context"
	"flag"
	"io"
	"log"
	"os"
	"testing"

	"github.com/ONSdigital/dp-cantabular-xlsx-exporter/config"
	"github.com/ONSdigital/dp-cantabular-xlsx-exporter/features/steps"
	cmponenttest "github.com/ONSdigital/dp-component-test"
	dplogs "github.com/ONSdigital/log.go/v2/log"
	"github.com/cucumber/godog"
	"github.com/cucumber/godog/colors"
)

const (
	mongoVersion     = "4.4.8"
	databaseName     = "filters"
	componentLogFile = "component-output.txt"
)

var componentFlag = flag.Bool("component", false, "perform component tests")

type ComponentTest struct {
	MongoFeature *cmponenttest.MongoFeature
}

func (f *ComponentTest) InitializeScenario(ctx *godog.ScenarioContext) {
	mongoAddr := f.MongoFeature.Server.URI()
	component := steps.NewComponent(mongoAddr)

	ctx.Before(func(ctx context.Context, sc *godog.Scenario) (context.Context, error) {
		if err := component.Reset(); err != nil {
			log.Panicf("unable to initialise scenario: %s", err)
		}
		return ctx, nil
	})

	ctx.After(func(ctx context.Context, sc *godog.Scenario, err error) (context.Context, error) {
		component.Close()
		return ctx, nil
	})

	component.RegisterSteps(ctx)
}

func (f *ComponentTest) InitializeTestSuite(ctx *godog.TestSuiteContext) {
	ctx.BeforeSuite(func() {
		f.MongoFeature = cmponenttest.NewMongoFeature(cmponenttest.MongoOptions{
			MongoVersion: mongoVersion,
			DatabaseName: databaseName,
		})
	})
	ctx.AfterSuite(func() {
		if err := f.MongoFeature.Close(); err != nil {
			log.Printf("failed to close mongo feature: %s", err)
		}
	})
}

func TestComponent(t *testing.T) {
	if *componentFlag {
		status := 0

		cfg, err := config.Get()
		if err != nil {
			t.Fatalf("failed to get service config: %s", err)
		}

		var output io.Writer = os.Stdout

		if cfg.ComponentTestUseLogFile {
			logfile, err := os.Create(componentLogFile)
			if err != nil {
				t.Fatalf("could not create logs file: %s", err)
			}

			defer func() {
				if err := logfile.Close(); err != nil {
					t.Fatalf("failed to close logs file: %s", err)
				}
			}()
			output = logfile

			dplogs.SetDestination(logfile, nil)
		}

		var opts = godog.Options{
			Output: colors.Colored(output),
			Format: "pretty",
			Paths:  flag.Args(),
		}

		f := &ComponentTest{}

		status = godog.TestSuite{
			Name:                 "feature_tests",
			ScenarioInitializer:  f.InitializeScenario,
			TestSuiteInitializer: f.InitializeTestSuite,
			Options:              &opts,
		}.Run()

		if status > 0 {
			t.Fail()
		}
	} else {
		t.Skip("component flag required to run component tests")
	}
}
