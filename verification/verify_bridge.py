from playwright.sync_api import sync_playwright

def verify_bridge_ui():
    with sync_playwright() as p:
        browser = p.chromium.launch()
        page = browser.new_page()
        try:
            print("Navigating to http://localhost:3000...")
            page.goto("http://localhost:3000")

            # Wait for initial load
            page.wait_for_selector("#dashboard")

            print("Clicking Wallet tab...")
            page.click("button[data-tab='wallet']")

            # Wait for Bridge Card
            print("Verifying Bridge Status...")
            page.wait_for_selector("text=Bridge Status")
            page.wait_for_selector("#bridge-network")

            # Verify data population (wait for network name)
            print("Waiting for data...")
            page.wait_for_selector("#bridge-network:not(:has-text('-'))", timeout=5000)

            # Take screenshot
            screenshot_path = "verification/bridge_status.png"
            page.screenshot(path=screenshot_path)
            print(f"Screenshot saved to {screenshot_path}")

        except Exception as e:
            print(f"Error: {e}")
            page.screenshot(path="verification/bridge_error.png")
        finally:
            browser.close()

if __name__ == "__main__":
    verify_bridge_ui()
