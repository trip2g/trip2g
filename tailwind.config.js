/** @type {import('tailwindcss').Config} */
module.exports = {
  content: ["./internal/case/**/*.qtpl"],
  // plugins: [
  //   require('@tailwindcss/typography'),
  //   require('daisyui'),
  // ],
  theme: {
    extend: {
      colors: {
        border: "hsl(var(--border))",
        input: "hsl(var(--input))",
        ring: "hsl(var(--ring))",
        background: "hsl(var(--background))",
        foreground: "hsl(var(--foreground))",
        primary: {
          DEFAULT: "hsl(var(--primary))",
          foreground: "hsl(var(--primary-foreground))",
        },
        secondary: {
          DEFAULT: "hsl(var(--secondary))",
          foreground: "hsl(var(--secondary-foreground))",
        },
        destructive: {
          DEFAULT: "hsl(var(--destructive))",
          foreground: "hsl(var(--destructive-foreground))",
        },
        muted: {
          DEFAULT: "hsl(var(--muted))",
          foreground: "hsl(var(--muted-foreground))",
        },
        accent: {
          DEFAULT: "hsl(var(--accent))",
          foreground: "hsl(var(--accent-foreground))",
        },
        popover: {
          DEFAULT: "hsl(var(--popover))",
          foreground: "hsl(var(--popover-foreground))",
        },
        card: {
          DEFAULT: "hsl(var(--card))",
          foreground: "hsl(var(--card-foreground))",
        },
      },
      borderRadius: {
        lg: `var(--radius)`,
        md: `calc(var(--radius) - 2px)`,
        sm: "calc(var(--radius) - 4px)",
      },
      typography: () => ({
        DEFAULT: {
          css: {
          },
        },
        lg: {
          css: {
            h3: {
              fontSize: '21px',
              lineHeight: '26px',
              fontWeight: 'bold',
            },
            'ul ul, ul ol, ol ul, ol ol': {
              marginTop: '0.2rem',
              marginBottom: '0.2rem',
            },
            ul: {
              marginTop: '0.2rem',
              marginBottom: '0.2rem',
            },
            li: {
              marginTop: '0.2rem',
              marginBottom: '0.2rem',
            },
          },
        },
      }),
    },
  },
}

