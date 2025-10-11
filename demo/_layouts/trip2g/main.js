// SideBar Handler
document.addEventListener("DOMContentLoaded", function () {

	// Dropdown Handler
	function dropdownHandler(element) {
		let single = element.getElementsByTagName("ul")[0];
		single.classList.toggle("hidden");
	}

	// Get all menu items and submenus
	const menuItems = document.querySelectorAll("[data-d2c-dropdown]");
	const subMenus = document.querySelectorAll("[data-d2c-dropdownItem]");
	menuItems.forEach((menuItem, index) => {
		menuItem.addEventListener("click", () => {
			const submenu = subMenus[index];
			submenu.classList.toggle("hidden");
		});
	});

	// Form Validation
	const forms = document.querySelectorAll(".validation");

	forms.forEach((form) => {
		const inputFuild = form.querySelectorAll(
			"input[required], select[required], textarea[required]"
		);

		inputFuild.forEach((input) => {
			input.addEventListener("focus", () => {
				removeError(input);
			});
			input.addEventListener("blur", () => {
				validateInput(input);
			});
		});

		form.addEventListener("submit", function (event) {
			event.preventDefault();

			let isValid = true;

			inputFuild.forEach((input) => {
				if (!validateInput(input)) {
					isValid = false;
					input.classList.add("invalid");
				} else {
					input.classList.remove("invalid");
				}
			});

			if (isValid) {
				form.submit();
			}
		});
	});

	// Validation function
	function validateInput(input) {
		const value = input.value.trim();
		const type = input.getAttribute("type");

		if (value === "") {
			setError(input, "Please enter a value");
			return false;
		}

		if (type === "") {
			setError(input, "Please set input");
			return false;
		}

		if (type === "text") {
			if (value.length < 0) {
				setError(input, "Please enter a value");
				return false;
			}
		}

		if (type === "email") {
			const emailPattern =
				/^[a-zA-Z0-9._-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,4}$/;
			if (!emailPattern.test(value)) {
				setError(input, "Please enter a valid email address.");
				return false;
			}
		}

		if (type === "password") {
			if (value.length < 6) {
				setError(input, "Password must be at least 6 characters.");
				return false;
			}
		}

		removeError(input);
		return true;
	}

	function setError(input, errorMessage) {
		input.classList.add("invalid-input");
		const errorElement = document.getElementById(`${input.id}-error`);
	}

	function removeError(input) {
		input.classList.remove("invalid-input");
		const errorElement = document.getElementById(`${input.id}-error`);
		if (errorElement) {
			errorElement.textContent = "";
		}
	}

	// Tab
	var buttons = document.querySelectorAll(".tab-button");
	buttons.forEach(function (button) {
		button.onclick = function () {
			var tabId = button.getAttribute("data-d2c-tab");
			openTab(tabId);
		};
	});

	function openTab(tabId) {
		var tabContents = document.querySelectorAll(".tab-content");
		tabContents.forEach(function (tabContent) {
			tabContent.classList.add("hidden");
		});

		var tabButtons = document.querySelectorAll(".tab-button");
		tabButtons.forEach(function (button) {
			button.classList.remove("active");
		});

		var tabElement = document.getElementById(tabId);
		if (tabElement) {
			tabElement.classList.remove("hidden");
		}

		var tabButton = document.querySelector(`[data-d2c-tab="${tabId}"]`);
		if (tabButton) {
			tabButton.classList.add("active");
		}
	}

	openTab("tab1");

	// Navbar toggler click event
	var Navbar = document.getElementById("mobile_view_nav");
	var openButton = document.getElementById("navToggoler");
	var closeButton = document.getElementById("navCloser");

	if (Navbar && openButton && closeButton) {
		// Function to open the Navbar
		function openNav() {
			Navbar.classList.remove("translate-x-full");
			Navbar.classList.add("translate-x-0");
		}

		// Function to close the Navbar
		function closeNav() {
			Navbar.classList.remove("translate-x-0");
			Navbar.classList.add("translate-x-full");
		}

		openButton.addEventListener("click", function () {
			openNav();
		});

		closeButton.addEventListener("click", function () {
			closeNav();
		});
	}

	// Clone navigation elements and append to mobile view
	document.querySelectorAll("#js-clone-nav").forEach(function (element) {
		const clonedNav = element.cloneNode(true);
		document.querySelector("#mobile_view_nav").appendChild(clonedNav);
	});
});


// // Preloader Js
// // Set initial opacity
$(".preloader").css("opacity", 1);

// Delay execution for 2 seconds
setTimeout(function() {
    // Set opacity to 0
    $(".preloader").css("opacity", 0);
    // After a short delay (for the fade-out effect to complete), set display to none
    setTimeout(function() {
        $(".preloader").css("display", "none");
    }, 400); // Adjust the delay to match the fade-out duration
}, 400);