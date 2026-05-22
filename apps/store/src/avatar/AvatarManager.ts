import * as THREE from "three";
import { GLTFLoader } from "three/examples/jsm/loaders/GLTFLoader.js";
import { DRACOLoader } from "three/examples/jsm/loaders/DRACOLoader.js";

// ── Procedural fallback: anatomically-shaped human body ──
function createProceduralBody(
  gender: "male" | "female",
  fat: number,
  muscle: number,
): THREE.Group {
  const isMale = gender === "male";
  const f = fat / 100;
  const m = muscle / 100;

  const sw = isMale ? 1.0 + m * 0.3 : 0.85 + m * 0.1;
  const hw = isMale ? 0.9 + f * 0.1 : 1.0 + f * 0.15;
  const wst = isMale ? 0.85 : 0.75 - f * 0.05;
  const chDepth = isMale ? 0.65 + m * 0.1 : 0.6 + f * 0.05;
  const bust = !isMale ? 0.3 + f * 0.1 : 0.0;
  const belly = 0.55 + f * 0.3;
  const butt = 0.25 + f * 0.15;

  // Body profile rings: [y, radiusX, radiusZ, frontOffset, backOffset]
  const rings: [number, number, number, number, number][] = [
    [0.0, 0.07, 0.06, 0, 0.03],
    [0.15, 0.08, 0.07, 0, 0.02],
    [0.3, 0.09, 0.08, 0, 0],
    [0.45, 0.1, 0.09, 0, 0],
    [0.52, 0.09, 0.08, 0, 0],
    [0.6, 0.11, 0.1, 0, 0.03],
    [0.7, 0.12, 0.11, 0, 0.05],
    [0.8, 0.13 * hw, 0.12 * hw, 0, butt],
    [0.85, 0.14 * hw, 0.13 * hw, 0, butt],
    [0.92, 0.12 * wst, 0.1 * wst, belly * 0.3, butt * 0.7],
    [1.0, 0.13 * wst, 0.11 * wst, belly * 0.5, butt * 0.5],
    [1.08, 0.16 * sw, 0.12 * chDepth, belly * 0.7, 0.08],
    [1.15, 0.18 * sw, 0.13 * chDepth, bust * 0.5, 0.05],
    [1.2, 0.19 * sw, 0.14 * chDepth, bust, 0.03],
    [1.25, 0.18 * sw, 0.13 * chDepth, bust * 0.7, 0.02],
    [1.3, 0.2 * sw, 0.12, 0, 0.02],
    [1.35, 0.17 * sw, 0.11, 0, 0.01],
    [1.4, 0.11, 0.1, 0, 0],
    [1.45, 0.08, 0.08, 0, 0],
  ];

  // Build mesh from rings
  const segs = 28;
  const verts: number[] = [];
  const idx: number[] = [];
  for (let ri = 0; ri < rings.length; ri++) {
    const [y, rx, rz, fo, bo] = rings[ri];
    for (let si = 0; si < segs; si++) {
      const a = (si / segs) * Math.PI * 2;
      let r = rx + (rz - rx) * Math.abs(Math.cos(a));
      r += fo * Math.max(0, -Math.cos(a)) + bo * Math.max(0, Math.cos(a));
      verts.push(Math.cos(a) * r, y, Math.sin(a) * r);
    }
  }
  for (let ri = 0; ri < rings.length - 1; ri++) {
    for (let si = 0; si < segs; si++) {
      const a = ri * segs + si,
        b = ri * segs + ((si + 1) % segs);
      const c = (ri + 1) * segs + si,
        d = (ri + 1) * segs + ((si + 1) % segs);
      idx.push(a, b, d, a, d, c);
    }
  }
  const bodyGeo = new THREE.BufferGeometry();
  bodyGeo.setAttribute("position", new THREE.Float32BufferAttribute(verts, 3));
  bodyGeo.setIndex(idx);
  bodyGeo.computeVertexNormals();

  const mat = new THREE.MeshStandardMaterial({
    color: "#44944A",
    metalness: 0.25,
    roughness: 0.4,
  });
  const matJoint = new THREE.MeshStandardMaterial({
    color: "#1a3d1f",
    metalness: 0.5,
    roughness: 0.3,
  });
  const jointGeo = new THREE.SphereGeometry(0.04, 12, 8);

  const group = new THREE.Group();
  group.add(new THREE.Mesh(bodyGeo, mat));

  // Head
  const head = new THREE.Mesh(new THREE.SphereGeometry(0.11, 24, 20), mat);
  head.position.set(0, 1.58, 0);
  head.scale.set(0.95, 1.1, 0.9);
  group.add(head);

  // Arms
  const armGeo = new THREE.CylinderGeometry(0.05, 0.04, 0.55, 16);
  [-1, 1].forEach((side) => {
    const armGrp = new THREE.Group();
    armGrp.position.set(side * 0.18 * (isMale ? 1.2 : 1.0), 1.25, 0);
    armGrp.add(new THREE.Mesh(jointGeo, matJoint));
    const upper = new THREE.Mesh(armGeo, mat);
    upper.position.set(side * 0.01, -0.3, 0.02);
    upper.rotation.set(0.15, 0, side * 0.08);
    armGrp.add(upper);
    const elbow = new THREE.Mesh(jointGeo, matJoint);
    elbow.position.set(side * 0.008, -0.58, 0.02);
    armGrp.add(elbow);
    group.add(armGrp);
  });

  // Legs
  const legGeo = new THREE.CylinderGeometry(0.09, 0.07, 0.65, 16);
  [-1, 1].forEach((side) => {
    const legGrp = new THREE.Group();
    legGrp.position.set(side * 0.08, 0.78, 0);
    const hip = new THREE.Mesh(jointGeo, matJoint);
    hip.scale.set(1.1, 1.1, 1.1);
    legGrp.add(hip);
    const upper = new THREE.Mesh(legGeo, mat);
    upper.position.set(0, -0.35, 0);
    legGrp.add(upper);
    const knee = new THREE.Mesh(jointGeo, matJoint);
    knee.position.set(0, -0.68, 0);
    legGrp.add(knee);
    group.add(legGrp);
  });

  return group;
}

// ── Avatar Manager ──
type Gender = "male" | "female";

interface AvatarParams {
  gender: Gender;
  height: number;
  weight: number;
  fat: number;
  muscle: number;
}

class AvatarManagerClass {
  private cache = new Map<string, THREE.Group>();
  private loader: GLTFLoader | null = null;

  private getLoader(): GLTFLoader {
    if (!this.loader) {
      this.loader = new GLTFLoader();
      const draco = new DRACOLoader();
      draco.setDecoderPath(
        "https://www.gstatic.com/draco/versioned/decoders/1.5.6/",
      );
      this.loader.setDRACOLoader(draco);
    }
    return this.loader;
  }

  async loadGLB(gender: Gender): Promise<THREE.Group | null> {
    // Try gender-specific model first, then combined
    const paths = [
      `/models/${gender}-base.glb`,
      `/models/male_and_female_body_base_mesh_free_to_download.glb`,
    ];

    for (const path of paths) {
      if (this.cache.has(path)) {
        const cached = this.cache.get(path)!;
        const model = cached.clone(true);
        // If combined model, extract gender-specific parts
        if (path.includes("male_and_female")) {
          return this.extractGender(model, gender) || model;
        }
        return model;
      }

      try {
        const gltf = await this.getLoader().loadAsync(path);
        this.cache.set(path, gltf.scene);
        const model = gltf.scene.clone(true);
        if (path.includes("male_and_female")) {
          return this.extractGender(model, gender) || model;
        }
        return model;
      } catch (e) {
        console.warn(`GLB not found: ${path}`);
        continue;
      }
    }

    console.warn("No GLB models found, using procedural fallback");
    return null;
  }

  // Try to extract gender-specific meshes from combined model
  private extractGender(
    scene: THREE.Group,
    gender: Gender,
  ): THREE.Group | null {
    const result = new THREE.Group();
    let found = false;

    scene.traverse((child) => {
      const name = child.name.toLowerCase();
      // Look for gender-specific meshes/bones
      if (child instanceof THREE.Mesh || child instanceof THREE.SkinnedMesh) {
        const isMalePart =
          name.includes("male") || name.includes("man") || name.includes("m_");
        const isFemalePart =
          name.includes("female") ||
          name.includes("woman") ||
          name.includes("f_");

        if (gender === "male" && isMalePart) {
          result.add(child.clone());
          found = true;
        } else if (gender === "female" && isFemalePart) {
          result.add(child.clone());
          found = true;
        } else if (!isMalePart && !isFemalePart) {
          // Neutral parts (shared)
          result.add(child.clone());
          found = true;
        }
      }
    });

    return found ? result : null;
  }

  createProcedural(gender: Gender, fat: number, muscle: number): THREE.Group {
    return createProceduralBody(gender, fat, muscle);
  }

  applyParams(model: THREE.Group, params: AvatarParams): void {
    const { height, weight } = params;
    const hNorm = (height - 150) / 50;
    const wNorm = (weight - 40) / 80;
    const scale = 0.75 + hNorm * 0.5;
    model.scale.setScalar(scale);

    // Apply weight via body scaling
    const bw = 0.7 + wNorm * 0.6;
    model.children.forEach((child) => {
      if (
        child instanceof THREE.Mesh &&
        child.geometry.type === "BufferGeometry"
      ) {
        const name = (child as any).name || "";
        if (name.includes("torso") || name.includes("body")) {
          child.scale.x = bw;
          child.scale.z = bw;
        }
      }
    });
  }
}

export const AvatarManager = new AvatarManagerClass();
export type { AvatarParams, Gender };
