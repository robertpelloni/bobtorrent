from playwright.sync_api import sync_playwright

def verify_topology_map():
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

            # Wait for Topology Canvas
            page.wait_for_selector("#topology-canvas")

            # Take screenshot
            screenshot_path = "verification/topology_map.png"
            page.screenshot(path=screenshot_path)
            print(f"Screenshot saved to {screenshot_path}")

        except Exception as e:
            print(f"Error: {e}")
            page.screenshot(path="verification/topology_error.png")
        finally:
            browser.close()

if __name__ == "__main__":
    verify_topology_map()
