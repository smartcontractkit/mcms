package mcms

// fakeWriter is a fake implementation of io.Writer.
type fakeWriter struct {
	n   int
	err error
}

// newFakeWriter returns a new Writer.
func newFakeWriter(n int, err error) *fakeWriter {
	return &fakeWriter{
		n:   n,
		err: err,
	}
}

// Write doesn't actually write anything, it just returns the values in the Writer.
func (w *fakeWriter) Write(p []byte) (n int, err error) {
	return w.n, w.err
}

// fakeSigner implements the signer interface for testing purposes
type fakeSigner struct {
	sigB []byte
	err  error
}

// newFakeSigner creates a new fakeSigner. The args provided will be returned when Sign is called.
func newFakeSigner(sigB []byte, err error) signer {
	return &fakeSigner{sigB: sigB, err: err}
}

// Sign implemnts the signer interface.
func (f *fakeSigner) Sign(payload []byte) ([]byte, error) {
	return f.sigB, f.err
}
