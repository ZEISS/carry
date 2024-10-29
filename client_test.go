package carry

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNew(t *testing.T) {
	c := New()
	assert.NotNil(t, c)
}

func TestBase64Encode(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{
			input:    "http://zeisss.com/",
			expected: "http://zeisss.com/",
		},
		{
			input:    "http://zeisss.com",
			expected: "http://zeisss.com",
		},
		{
			input:    "/123",
			expected: "/123",
		},
		{
			input:    "/123/",
			expected: "/123/",
		},
		{
			input:    "123",
			expected: "123",
		},
	}

	for _, test := range tests {
		c := New().Base(test.input)
		require.Equal(t, test.expected, c.rawURL)
	}
}

func TestMethod(t *testing.T) {
	tests := []struct {
		c        *Client
		expected string
	}{
		{
			c:        New().Get("http://zeisss.com/"),
			expected: "GET",
		},
		{
			c:        New().Post("http://zeisss.com/"),
			expected: "POST",
		},
		{
			c:        New().Put("http://zeisss.com/"),
			expected: "PUT",
		},
		{
			c:        New().Patch("http://zeisss.com/"),
			expected: "PATCH",
		},
		{
			c:        New().Delete("http://zeisss.com/"),
			expected: "DELETE",
		},
		{
			c:        New().Options("http://zeisss.com/"),
			expected: "OPTIONS",
		},
		{
			c:        New().Head("http://zeisss.com/"),
			expected: "HEAD",
		},
		{
			c:        New().Connect("http://zeisss.com/"),
			expected: "CONNECT",
		},
		{
			c:        New().Trace("http://zeisss.com/"),
			expected: "TRACE",
		},
	}

	for _, test := range tests {
		require.Equal(t, test.expected, test.c.method)
	}
}
