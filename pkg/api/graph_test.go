package api

import (
	"context"
	"reflect"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestMatches(t *testing.T) {
	var testCases = []struct {
		name    string
		first   StepLink
		second  StepLink
		matches bool
	}{
		{
			name:    "internal matches itself",
			first:   InternalImageLink(PipelineImageStreamTagReferenceRPMs),
			second:  InternalImageLink(PipelineImageStreamTagReferenceRPMs),
			matches: true,
		},
		{
			name:    "external matches itself",
			first:   ExternalImageLink(ImageStreamTagReference{Namespace: "ns", Name: "name", Tag: "latest"}),
			second:  ExternalImageLink(ImageStreamTagReference{Namespace: "ns", Name: "name", Tag: "latest"}),
			matches: true,
		},
		{
			name:    "rpm matches itself",
			first:   RPMRepoLink(),
			second:  RPMRepoLink(),
			matches: true,
		},
		{
			name:    "release images matches itself",
			first:   ReleaseImagesLink(LatestReleaseName),
			second:  ReleaseImagesLink(LatestReleaseName),
			matches: true,
		},
		{
			name:    "different internal do not match",
			first:   InternalImageLink(PipelineImageStreamTagReferenceRPMs),
			second:  InternalImageLink(PipelineImageStreamTagReferenceSource),
			matches: false,
		},
		{
			name:    "different external do not match",
			first:   ExternalImageLink(ImageStreamTagReference{Namespace: "ns", Name: "name", Tag: "latest"}),
			second:  ExternalImageLink(ImageStreamTagReference{Namespace: "ns", Name: "name", Tag: "other"}),
			matches: false,
		},
		{
			name:    "internal does not match external",
			first:   InternalImageLink(PipelineImageStreamTagReferenceRPMs),
			second:  ExternalImageLink(ImageStreamTagReference{Namespace: "ns", Name: "name", Tag: "latest"}),
			matches: false,
		},
		{
			name:    "internal does not match RPM",
			first:   InternalImageLink(PipelineImageStreamTagReferenceRPMs),
			second:  RPMRepoLink(),
			matches: false,
		},
		{
			name:    "internal does not match release images",
			first:   InternalImageLink(PipelineImageStreamTagReferenceRPMs),
			second:  ReleaseImagesLink(LatestReleaseName),
			matches: false,
		},
		{
			name:    "external does not match RPM",
			first:   ExternalImageLink(ImageStreamTagReference{Namespace: "ns", Name: "name", Tag: "latest"}),
			second:  RPMRepoLink(),
			matches: false,
		},
		{
			name:    "external does not match release images",
			first:   ExternalImageLink(ImageStreamTagReference{Namespace: "ns", Name: "name", Tag: "latest"}),
			second:  ReleaseImagesLink(LatestReleaseName),
			matches: false,
		},
		{
			name:    "RPM does not match release images",
			first:   RPMRepoLink(),
			second:  ReleaseImagesLink(LatestReleaseName),
			matches: false,
		},
	}

	for _, testCase := range testCases {
		if actual, expected := testCase.first.SatisfiedBy(testCase.second), testCase.matches; actual != expected {
			message := "not match"
			if testCase.matches {
				message = "match"
			}
			t.Errorf("%s: expected links to %s, but they didn't:\nfirst:\n\t%v\nsecond:\n\t%v", testCase.name, message, testCase.first, testCase.second)
		}
	}
}

type fakeStep struct {
	requires []StepLink
	creates  []StepLink
	name     string
}

func (f *fakeStep) Inputs() (InputDefinition, error) { return nil, nil }

func (f *fakeStep) Run(ctx context.Context) error { return nil }

func (f *fakeStep) Requires() []StepLink { return f.requires }
func (f *fakeStep) Creates() []StepLink  { return f.creates }
func (f *fakeStep) Name() string         { return f.name }
func (f *fakeStep) Description() string  { return f.name }

func (f *fakeStep) Provides() ParameterMap { return nil }

func TestBuildGraph(t *testing.T) {
	root := &fakeStep{
		requires: []StepLink{ExternalImageLink(ImageStreamTagReference{Namespace: "ns", Name: "base", Tag: "latest"})},
		creates:  []StepLink{InternalImageLink(PipelineImageStreamTagReferenceRoot)},
	}
	other := &fakeStep{
		requires: []StepLink{ExternalImageLink(ImageStreamTagReference{Namespace: "ns", Name: "base", Tag: "other"})},
		creates:  []StepLink{InternalImageLink("other")},
	}
	src := &fakeStep{
		requires: []StepLink{InternalImageLink(PipelineImageStreamTagReferenceRoot)},
		creates:  []StepLink{InternalImageLink(PipelineImageStreamTagReferenceSource)},
	}
	bin := &fakeStep{
		requires: []StepLink{InternalImageLink(PipelineImageStreamTagReferenceSource)},
		creates:  []StepLink{InternalImageLink(PipelineImageStreamTagReferenceBinaries)},
	}
	testBin := &fakeStep{
		requires: []StepLink{InternalImageLink(PipelineImageStreamTagReferenceSource)},
		creates:  []StepLink{InternalImageLink(PipelineImageStreamTagReferenceTestBinaries)},
	}
	rpm := &fakeStep{
		requires: []StepLink{InternalImageLink(PipelineImageStreamTagReferenceBinaries)},
		creates:  []StepLink{InternalImageLink(PipelineImageStreamTagReferenceRPMs)},
	}
	unrelated := &fakeStep{
		requires: []StepLink{InternalImageLink("other"), InternalImageLink(PipelineImageStreamTagReferenceRPMs)},
		creates:  []StepLink{InternalImageLink("unrelated")},
	}
	final := &fakeStep{
		requires: []StepLink{InternalImageLink("unrelated")},
		creates:  []StepLink{InternalImageLink("final")},
	}

	duplicateRoot := &fakeStep{
		requires: []StepLink{ExternalImageLink(ImageStreamTagReference{Namespace: "ns", Name: "base", Tag: "latest"})},
		creates:  []StepLink{InternalImageLink(PipelineImageStreamTagReferenceRoot)},
	}
	duplicateSrc := &fakeStep{
		requires: []StepLink{
			InternalImageLink(PipelineImageStreamTagReferenceRoot),
			InternalImageLink(PipelineImageStreamTagReferenceRoot),
		},
		creates: []StepLink{InternalImageLink("other")},
	}

	var testCases = []struct {
		name   string
		input  []Step
		output []*StepNode
	}{
		{
			name:  "basic graph",
			input: []Step{root, other, src, bin, testBin, rpm, unrelated, final},
			output: []*StepNode{{
				Step: root,
				Children: []*StepNode{{
					Step: src,
					Children: []*StepNode{{
						Step: bin,
						Children: []*StepNode{{
							Step: rpm,
							Children: []*StepNode{{
								Step: unrelated,
								Children: []*StepNode{{
									Step:     final,
									Children: []*StepNode{},
								}},
							}},
						}},
					}, {
						Step:     testBin,
						Children: []*StepNode{},
					}},
				}},
			}, {
				Step: other,
				Children: []*StepNode{{
					Step: unrelated,
					Children: []*StepNode{{
						Step:     final,
						Children: []*StepNode{},
					}},
				}},
			}},
		},
		{
			name:  "duplicate links",
			input: []Step{duplicateRoot, duplicateSrc},
			output: []*StepNode{{
				Step: duplicateRoot,
				Children: []*StepNode{{
					Step:     duplicateSrc,
					Children: []*StepNode{},
				}},
			}},
		},
	}

	for _, testCase := range testCases {
		if actual, expected := BuildGraph(testCase.input), testCase.output; !reflect.DeepEqual(actual, expected) {
			t.Errorf("%s: did not generate step graph as expected:\nwant:\n\t%v\nhave:\n\t%v", testCase.name, expected, actual)
		}
	}
}

func TestReleaseNames(t *testing.T) {
	var testCases = []string{
		LatestReleaseName,
		InitialReleaseName,
		"foo",
	}
	for _, name := range testCases {
		stream := ReleaseStreamFor(name)
		if !IsReleaseStream(stream) {
			t.Errorf("stream %s for name %s was not identified as a release stream", stream, name)
		}
		if actual, expected := ReleaseNameFrom(stream), name; actual != expected {
			t.Errorf("parsed name %s from stream %s, but it was created for name %s", actual, stream, expected)
		}
	}

}

func TestLinkForImage(t *testing.T) {
	var testCases = []struct {
		stream, tag string
		expected    StepLink
	}{
		{
			stream:   "pipeline",
			tag:      "src",
			expected: InternalImageLink(PipelineImageStreamTagReferenceSource),
		},
		{
			stream:   "pipeline",
			tag:      "rpms",
			expected: InternalImageLink(PipelineImageStreamTagReferenceRPMs),
		},
		{
			stream:   "stable",
			tag:      "installer",
			expected: ReleaseImagesLink(LatestReleaseName),
		},
		{
			stream:   "stable-initial",
			tag:      "cli",
			expected: ReleaseImagesLink(InitialReleaseName),
		},
		{
			stream:   "stable-whatever",
			tag:      "hyperconverged-cluster-operator",
			expected: ReleaseImagesLink("whatever"),
		},
		{
			stream:   "release",
			tag:      "latest",
			expected: ReleasePayloadImageLink(LatestReleaseName),
		},
		{
			stream: "crazy",
			tag:    "tag",
		},
	}
	for _, testCase := range testCases {
		if diff := cmp.Diff(LinkForImage(testCase.stream, testCase.tag), testCase.expected, Comparer()); diff != "" {
			t.Errorf("got incorrect link for %s:%s: %v", testCase.stream, testCase.tag, diff)
		}
	}
}
