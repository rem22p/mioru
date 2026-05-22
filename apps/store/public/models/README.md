# 3D Avatar Models

Place your GLB models here:

- `male-base.glb` — male humanoid base mesh
- `female-base.glb` — female humanoid base mesh

## Requirements

- Format: GLB (binary glTF)
- Polygons: 5,000–20,000 (web-optimized)
- Must be rigged (skeleton/bones) for future clothing fitting
- Morph targets recommended: height, weight, muscular, breast_size
- Clean topology, watertight, no holes
- Standing A-pose or T-pose
- Scale: ~1.7-1.8 units tall

## How to get models

### Free options:
1. **MakeHuman** (makehumancommunity.org) — generates realistic human models, export as glTF
2. **MB-Lab** (Blender addon) — more detailed, better topology
3. **Sketchfab** — search "human base mesh", filter by downloadable + CC license

### Quick start:
1. Download MakeHuman from makehumancommunity.org
2. Create male and female characters
3. Export as "glTF Binary (.glb)"
4. Place files here as `male-base.glb` and `female-base.glb`

## Fallback

If no GLB files are found, the app uses a procedural body generator.
The GLB models will be loaded automatically once placed here.
