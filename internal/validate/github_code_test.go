package validate

import "testing"

func TestIsGitHubSourceCodePaste(t *testing.T) {
	t.Parallel()
	code := `import React from 'react'
import { motion } from 'framer-motion'
export default function Landing() {
  const [x, setX] = useState(0);
  return <div className="hero">keitaro voluum binom tracker</div>;
}`
	if !IsGitHubSourceCodePaste(code, "Iphone-16 landing") {
		t.Fatal("expected frontend paste to match")
	}
	if IsGitHubSourceCodePaste("voluum postback failing after keitaro migration", "Help needed") {
		t.Fatal("expected buyer pain issue to pass")
	}
}
