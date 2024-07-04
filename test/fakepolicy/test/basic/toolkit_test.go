// Copyright Contributors to the Open Cluster Management project

package basic

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Additional Toolkit tests", func() {
	It("Config methods should override values, but preserve the original toolkit", func() {
		newTK := tk.
			WithEPoll("50ms").
			WithETimeout("3s").
			WithCPoll("500ms").
			WithCTimeout("2s")

		Expect(newTK.EventuallyPoll).To(Equal("50ms"))
		Expect(newTK.EventuallyTimeout).To(Equal("3s"))
		Expect(newTK.ConsistentlyPoll).To(Equal("500ms"))
		Expect(newTK.ConsistentlyTimeout).To(Equal("2s"))

		Expect(tk.EventuallyPoll).To(Equal("100ms"))
		Expect(tk.EventuallyTimeout).To(Equal("1s"))
		Expect(tk.ConsistentlyPoll).To(Equal("100ms"))
		Expect(tk.ConsistentlyTimeout).To(Equal("1s"))
	})

	It("Kubectl should return error outputs", func() {
		output, err := tk.Kubectl("get", "node", "nonexist")
		Expect(output).To(BeEmpty())
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("not found"))
	})
})
