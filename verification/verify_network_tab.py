from playwright.sync_api import sync_playwright

def verify_network_tab():
    with sync_playwright() as p:
        browser = p.chromium.launch()
        page = browser.new_page()
        try:
            print("Navigating to http://localhost:3000...")
            page.goto("http://localhost:3000")

            # Wait for initial load
            page.wait_for_selector("#dashboard")

            print("Clicking Network tab...")
            page.click("button[data-tab='network']")

            # Wait for Network tab content
            page.wait_for_selector("#network")

            # Wait for data population (table row shouldn't say "Loading...")
            # We wait for "DHT (UDP)" to appear in the table
            print("Waiting for data...")
            page.wait_for_selector("text=DHT (UDP)", timeout=5000)

            # Take screenshot
            screenshot_path = "verification/network_tab.png"
            page.screenshot(path=screenshot_path)
            print(f"Screenshot saved to {screenshot_path}")

        except Exception as e:
            print(f"Error: {e}")
            page.screenshot(path="verification/error.png")
        finally:
            browser.close()

if __name__ == "__main__":
    verify_network_tab()
