from playwright.sync_api import sync_playwright

def verify_dashboard_resources():
    with sync_playwright() as p:
        browser = p.chromium.launch()
        page = browser.new_page()
        try:
            print("Navigating to http://localhost:3000...")
            page.goto("http://localhost:3000")

            # Wait for initial load
            page.wait_for_selector("#dashboard")

            # Wait for Resources Card
            print("Verifying Resources Card...")
            page.wait_for_selector("text=Load Level:")
            page.wait_for_selector("text=Memory:")
            page.wait_for_selector("text=AI Advice:")

            # Verify data population (wait for non-default value)
            print("Waiting for data...")
            page.wait_for_selector("#dash-load-level:not(:has-text('-'))", timeout=5000)

            # Take screenshot
            screenshot_path = "verification/dashboard_resources.png"
            page.screenshot(path=screenshot_path)
            print(f"Screenshot saved to {screenshot_path}")

        except Exception as e:
            print(f"Error: {e}")
            page.screenshot(path="verification/dashboard_error.png")
        finally:
            browser.close()

if __name__ == "__main__":
    verify_dashboard_resources()
