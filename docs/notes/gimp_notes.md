## Extracting Parts of an Image

# Extracting a Button from a Single-Layer Image in GIMP  
Using **Method 1 (Fuzzy Select / Magic Wand)** and **Method 2 (Select by Color)**

This guide explains how to isolate a UI button from a single-layer image when the background is a solid or near-solid color.

---

# Method 1 — Fuzzy Select (Magic Wand)

Best used when the background is mostly uniform and the button border is clearly defined.

## Steps

1. **Open your image**  
   `File → Open`

2. **Add an alpha channel**  
   `Layer → Transparency → Add Alpha Channel`  
   *(Skip if already enabled.)*

3. **Select the Fuzzy Select Tool**  
   - Click the magic wand icon  
   - Or press `U`

4. **Set Tool Options**
   - Mode: `Replace the current selection`
   - Antialiasing: **on**
   - Feather edges: **off**
   - Threshold: **20–30** (adjust depending on result)
   - Sample merged: **off** (unless using multiple layers)
u
5. **Click the background**  
   The background should become selected.  
   - If too much of the button is selected → lower Threshold  
   - If too little is selected → raise Threshold

6. **Invert the selection**  
   `Select → Invert`  
   Now the button is selected.

7. **Optional — Refine the selection**
   - Shrink selection: `Select → Shrink → 1–2 px`  
     *(Prevents background fringe from being included.)*
   - Feather edges: `Select → Feather → 1 px`  
     *(Smooths edge transition.)*

8. **Copy to a new layer**
    - Edit → Copy
    - Edit → Paste As → New Layer

    or paste then choose `Layer → To New Layer`.

9. **Hide or delete the original layer**
Click the eye icon next to the original layer.

10. **Inspect and clean edges**
 - Use the **Eraser Tool** for stray pixels  
 - Or use **Quick Mask** (`Shift+Q`) for precise selection editing

11. **Export the isolated button**
 `File → Export As… → PNG`

---

# Method 2 — Select by Color

Best used when the background has small variations or you want to target a specific color across the entire image.

## Steps

1. **Open your image**  
`File → Open`

2. **Add an alpha channel**  
`Layer → Transparency → Add Alpha Channel`

3. **Select the “Select by Color Tool”**  
- Click the three-color-squares icon  
- Or press `Shift+O`

4. **Set Tool Options**
- Threshold: **20–40**  
- Antialiasing: **on**
- Feather edges: **off**
- Sample merged: **off**

5. **Click on the background**  
GIMP will select all pixels of similar color.  
- If button pixels get selected → lower Threshold  
- If background is incomplete → raise Threshold

6. **Invert the selection**  
`Select → Invert`  
This selects the button instead of the background.

7. **Optional — Clean up the selection**
- `Select → Shrink → 1–2 px`
- `Select → Feather → 1 px`

8. **Copy to a new layer**
    - Edit → Copy
    - Edit → Paste As → New Layer

    
9. **Hide or delete the original layer**

10. **Clean up using Quick Mask or a Layer Mask**
 - Quick Mask: `Shift+Q`  
 - Layer Mask: Right-click layer → `Add Layer Mask (Selection)`

11. **Export as a transparent PNG**
 `File → Export As… → PNG`

---

# Recommended Starting Values

- Fuzzy Select Threshold: **20–30**
- Select by Color Threshold: **20–40**
- Shrink: **1–2 px**
- Feather: **0.5–2 px**

---

# Extra Tips

- Use Quick Mask (`Shift+Q`) to fine-tune selections by painting.
- If a dark halo appears, shrink the selection by 1 px and delete the fringe.
- Save a `.xcf` file if you want to keep layers editable for later.

---

## Resizing an Image

# How to Change the Width and Height (in Pixels) of an Image in GIMP

Follow these steps to resize an image so that its pixel dimensions match the values shown in Windows Explorer → Properties → Details.

---

## 1. Open Your Image
1. Launch **GIMP**.
2. Go to **File → Open…**
3. Select your PNG (or other image file).

---

## 2. Open the Resize Dialog
GIMP provides a dedicated window for changing pixel dimensions.

1. In the top menu, go to:  
   **Image → Scale Image…**

---

## 3. Set the New Pixel Dimensions
In the **Scale Image** window:

1. Under **Image Size**, locate:
   - **Width**
   - **Height**
2. Click the **chain link icon** to toggle between:
   - **Linked** (maintains aspect ratio)
   - **Unlinked** (lets you freely change width/height independently)
3. Enter your desired pixel values.

---

## 4. Choose Interpolation Method (Optional)
Below the size fields is **Interpolation** — this determines how pixel information is handled during resizing.

Recommended settings:
- **Cubic** — good quality (default)
- **NoHalo** or **LoHalo** — best for UI assets or pixel-art–like elements
- **None** — only for true pixel art scaling with clean edges

Select the method that fits your image.

---

## 5. Apply the Resize
1. Click **Scale**.
2. Your image is now resized to the exact width and height in pixels that you specified.

---

## 6. Export the Image
To save your resized image:

1. Go to **File → Export As…**
2. Choose a filename (keep PNG for UI assets).
3. Click **Export**.
4. Confirm PNG settings when prompted.

---

## Summary
To change pixel dimensions in GIMP:
- Use **Image → Scale Image…**
- Enter new **Width** and **Height**
- Adjust interpolation if needed
- Export as PNG when done

Your image will now match the dimensions shown in Windows Explorer.
