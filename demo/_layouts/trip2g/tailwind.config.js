/** @type {import('tailwindcss').Config} */
module.exports = {
	mode: "jit",
	content: ["*.html"],
	theme: {
		screens: {
			sm: "576px",
			md: "768px",
			lg: "992px",
			xl: "1200px",
			xxl: "1500px",
		},
		container: {
			center: true,
			padding: "1rem",
		},
		colors: {
			primary: "#774AA4",
			"primary-1": "#E4DBED",
			"primary-2": "#F1EDF6",
			"primary-3": "#FCFAFF",
			secondary: "#37234B",
			info: "#4f4954",
			"info-1": "#7B7B7B",
			"info-2": "#C9C7CB",
			success: "#34AF3E",
			"success-1": "#E8F6EA",
			danger: "#CB3448",
			"danger-1": "#F9E8EB",
			warning: "#E6B800",
			"warning-1": "#FFF5CC",
			white: "#FFFFFF",
			global: "#AFA9B6",
			transparent: "#00000000",
		},
		fontFamily: {
			sans: ["Open Sans", "sans-serif"],
		},
		extend: {
			borderRadius: {
				std: "0.4rem",
				"std-1/2": "0.2rem",
			},
			boxShadow: {
				std: "0px 0px 24px 0px rgba(0, 0, 0, 0.05)",
			},
		},
	},
	plugins: [require('@tailwindcss/typography')],
};
