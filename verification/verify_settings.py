from playwright.sync_api import sync_playwright

def verify_settings_tab():
    with sync_playwright() as p:
        browser = p.chromium.launch()
        page = browser.new_page()
        try:
            print("Navigating to http://localhost:3000...")
            page.goto("http://localhost:3000")

            # Wait for initial load
            page.wait_for_selector("#dashboard")

            print("Clicking Settings tab...")
            page.click("button[data-tab='settings']")

            # Wait for Settings content
            page.wait_for_selector("#settings")

            # Verify fields
            print("Verifying Settings Fields...")
            page.wait_for_selector("#conf-name")
            page.wait_for_selector("#conf-clearnet")

            # Change a setting
            print("Changing Node Name...")
            page.fill("#conf-name", "Test Node Modified")

            # Save
            print("Saving...")
            page.click("#btn-save-config")

            # Verify status message
            page.wait_for_selector("text=Saved!", timeout=5000)

            # Take screenshot
            screenshot_path = "verification/settings_tab.png"
            page.screenshot(path=screenshot_path)
            print(f"Screenshot saved to {screenshot_path}")

        except Exception as e:
            print(f"Error: {e}")
            page.screenshot(path="verification/settings_error.png")
        finally:
            browser.close()

if __name__ == "__main__":
    verify_settings_tab()
