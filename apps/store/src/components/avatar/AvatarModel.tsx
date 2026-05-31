/// <reference types="@react-three/fiber" />
import { useEffect, useRef, useState } from "react";
import * as THREE from "three";
import { AvatarManager, type Gender } from "@/avatar/AvatarManager";

interface AvatarModelProps {
  params: {
    gender: "male" | "female";
    height: number;
    weight: number;
    fat: number;
    muscle: number;
  };
  wireframe?: boolean;
}

export default function AvatarModel({
  params,
  wireframe = false,
}: AvatarModelProps) {
  const groupRef = useRef<THREE.Group>(null);
  const modelRef = useRef<THREE.Group | null>(null);
  const [ready, setReady] = useState(false);

  const { gender, height, weight, fat, muscle } = params;

  useEffect(() => {
    let cancelled = false;

    async function init() {
      // Try loading GLB first
      const glbModel = await AvatarManager.loadGLB(gender as Gender);

      if (!cancelled) {
        if (glbModel) {
          modelRef.current = glbModel;
        } else {
          // Fallback to procedural
          modelRef.current = AvatarManager.createProcedural(
            gender as Gender,
            fat,
            muscle,
          );
        }

        if (groupRef.current) {
          // Remove old model
          while (groupRef.current.children.length > 0) {
            groupRef.current.remove(groupRef.current.children[0]);
          }
          groupRef.current.add(modelRef.current);
        }
        setReady(true);
      }
    }

    init();
    return () => {
      cancelled = true;
    };
  }, [gender]); // Reload only when gender changes

  // Apply parameter updates to existing model
  useEffect(() => {
    if (modelRef.current) {
      AvatarManager.applyParams(modelRef.current, {
        gender: gender as Gender,
        height,
        weight,
        fat,
        muscle,
      });
    }
  }, [height, weight, fat, muscle, gender]);

  // Toggle wireframe on all meshes
  useEffect(() => {
    if (modelRef.current) {
      modelRef.current.traverse((child) => {
        if (child instanceof THREE.Mesh && child.material) {
          const materials = Array.isArray(child.material)
            ? child.material
            : [child.material];
          materials.forEach((mat) => {
            if (mat instanceof THREE.MeshStandardMaterial) {
              mat.wireframe = wireframe;
            }
          });
        }
      });
    }
  }, [wireframe]);

  return <group ref={groupRef} />;
}
