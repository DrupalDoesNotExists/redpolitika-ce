"use client";

import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import MarkdownPage from "@/components/MarkdownPage";

const queryClient = new QueryClient();

export default function Page({ params }: { params: { slug: string } }) {
  return (
    <QueryClientProvider client={queryClient}>
      <MarkdownPage slug={params.slug} />
    </QueryClientProvider>
  );
}
