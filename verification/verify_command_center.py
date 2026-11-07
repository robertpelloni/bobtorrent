from playwright.sync_api import sync_playwright
import time

def verify_command_center():
    with sync_playwright() as p:
        browser = p.chromium.launch()
        page = browser.new_page(viewport={"width": 1400, "height": 900})
        try:
            print("Navigating to http://localhost:8000...")
            page.goto("http://localhost:8000")

            # Wait for main header and status bar
            page.wait_for_selector(".header-left h1")
            page.wait_for_selector("#dash-dht")
            page.wait_for_selector("#section-overview")

            # 1. Overview Screenshot
            page.click("button.nav-tab-btn[data-target='section-overview']")
            page.wait_for_selector("#section-overview.active")
            time.sleep(0.5)
            page.screenshot(path="verification/command_center_overview.png", full_page=True)

            # 2. Publish Screenshot
            page.click("button.nav-tab-btn[data-target='section-publish']")
            page.wait_for_selector("#section-publish.active")
            time.sleep(0.5)
            page.screenshot(path="verification/command_center_publish.png", full_page=True)

            # 3. Mesh Screenshot
            page.click("button.nav-tab-btn[data-target='section-mesh']")
            page.wait_for_selector("#section-mesh.active")
            time.sleep(0.5)
            page.screenshot(path="verification/command_center_mesh.png", full_page=True)

            # 4. Identity Screenshot
            page.click("button.nav-tab-btn[data-target='section-identity']")
            page.wait_for_selector("#section-identity.active")
            time.sleep(0.5)
            page.screenshot(path="verification/command_center_identity.png", full_page=True)

            # 5. Lattice Screenshot
            page.click("button.nav-tab-btn[data-target='section-lattice']")
            page.wait_for_selector("#section-lattice.active")
            time.sleep(0.5)
            page.screenshot(path="verification/command_center_lattice.png", full_page=True)

            print("All navigation tabs verified and captured successfully.")

        except Exception as e:
            print(f"Verification error: {e}")
            page.screenshot(path="verification/command_center_error.png")
            raise e
        finally:
            browser.close()

if __name__ == "__main__":
    verify_command_center()
