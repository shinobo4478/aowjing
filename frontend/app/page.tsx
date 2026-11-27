"use client";

import { useEffect, useState } from "react";
import { Spin } from "antd";
import { listProfiles } from "@/lib/api";
import Home from "./Home";
import GettingStarted from "./GettingStarted";

/**
 * Home: the quick summary once there is a profile to report on, the four getting-
 * started steps before that.
 */
export default function HomePage() {
  const [hasProfile, setHasProfile] = useState<boolean | null>(null);

  useEffect(() => {
    listProfiles()
      .then((p) => setHasProfile(p.length > 0))
      // If the list cannot be read, the steps are the safe thing to show.
      .catch(() => setHasProfile(false));
  }, []);

  if (hasProfile === null) {
    return (
      <div style={{ padding: 48, textAlign: "center" }}>
        <Spin />
      </div>
    );
  }
  return hasProfile ? <Home /> : <GettingStarted />;
}
