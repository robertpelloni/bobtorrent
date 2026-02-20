from playwright.sync_api import sync_playwright

def verify_peers_ui():
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

            # Wait for Peers table
            page.wait_for_selector("#peers-table")

            # Verify table header
            print("Verifying Peers Table...")
            page.wait_for_selector("text=Peer ID")
            page.wait_for_selector("text=Score")

            # Take screenshot
            screenshot_path = "verification/peers_table.png"
            page.screenshot(path=screenshot_path)
            print(f"Screenshot saved to {screenshot_path}")

        except Exception as e:
            print(f"Error: {e}")
            page.screenshot(path="verification/peers_error.png")
        finally:
            browser.close()

if __name__ == "__main__":
    verify_peers_ui()
