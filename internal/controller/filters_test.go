package controller

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("EvaluateFilters", func() {
	Context("excluded only", func() {
		excluded := Filters{
			Exact:  []string{"exc"},
			Prefix: []string{"exc-pre"},
			Suffix: []string{"exc-suf"},
		}

		It("should return false when excluded matches", func() {
			Expect(EvaluateFilters(excluded, Filters{}, "exc")).To(BeFalse())
			Expect(EvaluateFilters(excluded, Filters{}, "exc-pre-test")).To(BeFalse())
			Expect(EvaluateFilters(excluded, Filters{}, "test-exc-suf")).To(BeFalse())
		})

		It("should return true when excluded does not match", func() {
			Expect(EvaluateFilters(excluded, Filters{}, "test")).To(BeTrue())
			Expect(EvaluateFilters(excluded, Filters{}, "ex")).To(BeTrue())
			Expect(EvaluateFilters(excluded, Filters{}, "re")).To(BeTrue())
			Expect(EvaluateFilters(excluded, Filters{}, "uf")).To(BeTrue())
		})
	})

	Context("included only", func() {
		included := Filters{
			Exact:  []string{"inc"},
			Prefix: []string{"inc-pre"},
			Suffix: []string{"inc-suf"},
		}

		It("should return true when included matches", func() {
			Expect(EvaluateFilters(Filters{}, included, "inc")).To(BeTrue())
			Expect(EvaluateFilters(Filters{}, included, "inc-pre-test")).To(BeTrue())
			Expect(EvaluateFilters(Filters{}, included, "test-inc-suf")).To(BeTrue())
		})

		It("should return false when included does not match", func() {
			Expect(EvaluateFilters(Filters{}, included, "test")).To(BeFalse())
			Expect(EvaluateFilters(Filters{}, included, "in")).To(BeFalse())
			Expect(EvaluateFilters(Filters{}, included, "re")).To(BeFalse())
			Expect(EvaluateFilters(Filters{}, included, "uf")).To(BeFalse())
		})
	})

	Context("exclude and include", func() {
		excluded := Filters{
			Exact:  []string{"exc"},
			Prefix: []string{"exc-pre"},
			Suffix: []string{"exc-suf"},
		}

		included := Filters{
			Exact:  []string{"inc", "exc"},
			Prefix: []string{"inc-pre", "exc-pre"},
			Suffix: []string{"inc-suf", "exc-suf"},
		}

		It("should return true when excluded does not match and included matches", func() {
			Expect(EvaluateFilters(excluded, included, "inc")).To(BeTrue())
			Expect(EvaluateFilters(excluded, included, "inc-pre-test")).To(BeTrue())
			Expect(EvaluateFilters(excluded, included, "test-inc-suf")).To(BeTrue())
		})

		It("should return false when excluded does not match and included does not match", func() {
			Expect(EvaluateFilters(excluded, included, "test")).To(BeFalse())
			Expect(EvaluateFilters(excluded, included, "in")).To(BeFalse())
			Expect(EvaluateFilters(excluded, included, "re")).To(BeFalse())
			Expect(EvaluateFilters(excluded, included, "uf")).To(BeFalse())
		})

		It("should return false when excluded matches and included matches", func() {
			Expect(EvaluateFilters(excluded, included, "exc")).To(BeFalse())
			Expect(EvaluateFilters(excluded, included, "exc-pre-test")).To(BeFalse())
			Expect(EvaluateFilters(excluded, included, "test-exc-suf")).To(BeFalse())
		})
	})
})
