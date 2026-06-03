import os
import json
from openai import OpenAI

curl http://localhost:1234/api/v1/chat \
  -H "Content-Type: application/json" \
  -d '{
    "model": "qwen/qwen3.5-9b",
    "system_prompt": "You answer only in rhymes.",
    "input": "What is your favorite color?"
}'



# ==========================================
# API CONFIGURATION MATRIX
# ==========================================
# To use a mainstream model (OpenAI):
#   - BASE_URL = "https://api.openai.com/v1"
# http://localhost:1234/api/v1/chat
#   - MODEL_NAME = "gpt-4o"
#
# To use an uncensored/open model (via OpenRouter, DeepSeek, or LM Studio):
#   - OpenRouter BASE_URL = "https://openrouter.ai/api/v1"
#   - OpenRouter MODEL_NAME = "meta-llama/llama-3-70b-instruct" (or an uncensored variant)
#   - Local LM Studio BASE_URL = "http://localhost:1234/v1"

API_KEY = os.environ.get("LLM_API_KEY", "YOUR_API_KEY_HERE")
BASE_URL = "http://localhost:1234/api/v1/chat"  # Change this according to your provider
MODEL_NAME = "Dirty-Muse-Writer-v01-Uncensored-Erotica-NSFW-i1-GGUF"

client = OpenAI(api_key=API_KEY, base_url=BASE_URL)

def call_llm(system_prompt, user_prompt, JSON_mode=False):
    """Helper wrapper to handle text or structured JSON generation."""
    try:
        response_format = {"type": "json_object"} if JSON_mode else None
        
        response = client.chat.completions.create(
            model=MODEL_NAME,
            messages=[
                {"role": "system", "content": system_prompt},
                {"role": "user", "content": user_prompt}
            ],
            response_format=response_format,
            temperature=0.75
        )
        return response.choices[0].message.content.strip()
    except Exception as e:
        print(f"\n[!] API Error encountered: {e}")
        return None

# ==========================================
# CORE PIPELINE EXECUTION
# ==========================================

def run_ai_publishing_pipeline():
    print("=== INITIALIZING AI-POWERED ROMANTASY BOOK ARCHITECT ===")
    
    # -------------------------------------------------------------
    # PHASE 1: GENERATE STAGE-1 PLOT AND SYSTEM METADATA
    # -------------------------------------------------------------
    print("\n[+] Phase 1: Contacting LLM to generate elite Romantasy core concept...")
    plot_system = (
        "You are an expert commercial acquisition editor specializing in high-performing "
        "BookTok Romantasy fiction. You create highly addictive, trope-heavy concepts."
    )
    plot_user = (
        "Generate a core high-stakes Romantasy book pitch. It must include: "
        "1. Unique Female Main Character (FMC) description "
        "2. Morally grey Male Main Character (MMC) description "
        "3. A complex magic system name/premise "
        "4. High stakes world-level threat "
        "5. Two dominant tropes (e.g., Fated Mates, Enemies-to-Lovers, Who Did This To You?)."
    )
    
    core_concept = call_llm(plot_system, plot_user)
    if not core_concept: return
    print("\n--- CORE CONCEPT GENERATED ---")
    print(core_concept)
    
    # -------------------------------------------------------------
    # PHASE 2: GENERATE TABLE OF CONTENTS (STRUCTURED)
    # -------------------------------------------------------------
    print("\n[+] Phase 2: Generating structural Table of Contents based on the concept...")
    toc_system = (
        "You are a narrative architect. You translate a book concept into a rigid "
        "10-chapter structured narrative framework. You must output raw JSON format ONLY."
    )
    toc_user = (
        f"Based on this concept:\n{core_concept}\n\n"
        "Generate a 10-chapter Table of Contents formatted exactly as a JSON dictionary "
        "with a key named 'chapters'. Each item in 'chapters' must be an object containing:\n"
        "- 'chapter_number': integer\n"
        "- 'chapter_title': A dramatic Romantasy title (e.g., 'A Court of Ash and Diamonds')\n"
        "- 'narrative_focus': One sentence explaining what emotional/plot shift happens."
        "\nDo not return any markdown markdown wrapper except the raw JSON string."
    )
    
    toc_raw = call_llm(toc_system, toc_user, JSON_mode=True)
    if not toc_raw: return
    
    try:
        toc_data = json.loads(toc_raw)
    except Exception:
        print("[!] Failed to parse JSON framework. Printing raw text response instead:")
        print(toc_raw)
        return

    print("\n--- TABLE OF CONTENTS GENERATED ---")
    for ch in toc_data.get("chapters", []):
        print(f"Chapter {ch['chapter_number']}: {ch['chapter_title']}")
        print(f"  └─ Focus: {ch['narrative_focus']}")

    # -------------------------------------------------------------
    # PHASE 3: COMPILING CHAPTER DETAIL AND IMAGE DIRECTION
    # -------------------------------------------------------------
    print("\n[+] Phase 3: Fleshing out micro-summaries and production artwork guidelines...")
    
    # We will pick a key transition chapter dynamically to demonstrate execution speed
    sample_chapter = toc_data.get("chapters", [])[4] # Chapter 5 (Midpoint)
    
    chapter_system = (
        "You are an AI writing assistant and production designer. Your job is to extract an "
        "outline chunk and deliver deep writing instructions and visual prompts for text-to-image engines."
    )
    chapter_user = (
        f"Given the book core concept:\n{core_concept}\n\n"
        f"And this specific chapter detail:\n"
        f"Chapter {sample_chapter['chapter_number']}: {sample_chapter['chapter_title']}\n"
        f"Focus: {sample_chapter['narrative_focus']}\n\n"
        "Please provide:\n"
        "1. A detailed narrative beat paragraph for the AI writer to generate text from.\n"
        "2. A highly descriptive prompt for Midjourney/Stable Diffusion to create an illustration inside this chapter."
    )
    
    chapter_blueprint = call_llm(chapter_system, chapter_user)
    print(f"\n--- BLUEPRINT SAMPLE (CHAPTER {sample_chapter['chapter_number']}) ---")
    print(chapter_blueprint)

    # -------------------------------------------------------------
    # PHASE 4: PACKAGING THE PACKAGING (COVER & BLURB)
    # -------------------------------------------------------------
    print("\n[+] Phase 4: Finalizing physical production and marketing spec...")
    cover_system = (
        "You are a senior graphic director and copywriting expert working for a DTC publishing company."
    )
    cover_user = (
        f"Using this core book property:\n{core_concept}\n\n"
        "Provide a high-conversion specification sheet featuring:\n"
        "1. FRONT COVER ASSET SPEC: Art direction, color grading palette, font recommendations.\n"
        "2. BACK COVER BLURB: A highly sensational hook, body paragraph, and dramatic tagline optimized for social media sales."
    )
    
    cover_spec = call_llm(cover_system, cover_user)
    print("\n--- BRANDING AND COVER SPECIFICATIONS ---")
    print(cover_spec)
    
    print("\n=== PIPELINE RUN COMPLETE. SYSTEM READY TO WRITE PROSE ===")

if __name__ == "__main__":
    if API_KEY == "YOUR_API_KEY_HERE":
        print("[!] Action Required: Replace 'YOUR_API_KEY_HERE' string with your actual endpoint token.")
    else:
        run_ai_publishing_pipeline()