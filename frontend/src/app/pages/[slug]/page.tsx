"use client";

import { use } from "react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import MarkdownPage from "@/components/MarkdownPage";

const queryClient = new QueryClient();

export default function Page({ params }: { params: Promise<{ slug: string }> }) {
  const { slug } = use(params);
  return (
    <QueryClientProvider client={queryClient}>
      <MarkdownPage slug={slug} />
    </QueryClientProvider>
  );
}
