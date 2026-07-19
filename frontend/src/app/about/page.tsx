"use client";

import { useQuery } from "@tanstack/react-query";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { fetchVersion } from "@/lib/api";

const queryClient = new QueryClient();

function AboutContent() {
  const { data: version, isLoading, error } = useQuery({
    queryKey: ["version"],
    queryFn: fetchVersion,
    staleTime: 60 * 60 * 1000,
  });

  return (
    <div className="mx-auto max-w-[980px] px-8 editor-area">
      <h1 className="font-serif text-[28px] leading-[36px] mb-9 tracking-[-.01em]">
        О программе
      </h1>

      <article className="max-w-[640px] space-y-8 text-[17px] leading-[30px] text-[#1a1a1a]">
        <p>
          <span className="font-serif text-[22px] leading-[30px] tracking-[-.01em]">
            Редполитика
            <sup className="text-terra text-[0.55em] font-semibold leading-none align-super ml-[1px]">
              β
            </sup>
          </span>{" "}
          — сервис проверки и правки текста по правилам редакционной политики.
          Концептуально это «Главред, но для любой редполитики»: правила живут
          в YAML на диске, а не зашиты в код продукта.
        </p>

        <section>
          <h2 className="mb-3 font-serif text-[20px] leading-[28px] tracking-[-.01em]">
            Как устроена проверка
          </h2>
          <p className="mb-4 text-[#6b645a]">
            Вы пишете текст в редакторе — движок подсвечивает нарушения и
            предлагает правки. По ходу набора считаются два независимых балла
            от 0 до 10:
          </p>
          <ul className="list-disc space-y-2 pl-5 text-[#6b645a]">
            <li>
              <span className="text-[#1a1a1a]">чистота</span> — канцелярит,
              слова-паразиты, грубая и лишняя лексика;
            </li>
            <li>
              <span className="text-[#1a1a1a]">читаемость</span> — тяжёлые
              конструкции, длина фраз, то, что мешает читать.
            </li>
          </ul>
          <p className="mt-4 text-[#6b645a]">
            Простые правила (regex, списки слов) выполняются прямо в браузере
            и дают мгновенную подсветку. Сложные — с LLM, NER, POS или плагинами —
            считает сервер; результат приходит по WebSocket.
          </p>
        </section>

        <section>
          <h2 className="mb-3 font-serif text-[20px] leading-[28px] tracking-[-.01em]">
            Правила и плагины
          </h2>
          <p className="text-[#6b645a]">
            Редполитика собирается слоями: базовый набор, проектный и
            переопределения. Файлы YAML мержатся по стабильному{" "}
            <span className="font-mono text-[15px] text-[#1a1a1a]">id</span>{" "}
            правила. Тяжёлые стадии и интеграции выносятся в плагины
            (go-plugin / gRPC): ядро знает только точки расширения, а не
            «классы» плагинов.
          </p>
        </section>

        <section>
          <h2 className="mb-3 font-serif text-[20px] leading-[28px] tracking-[-.01em]">
            Community Edition
          </h2>
          <p className="text-[#6b645a]">
            Эта сборка — open-core ядро для self-host: один проект, без
            авторизации и мультитенантности. Enterprise Edition надстраивается
            плагинами (SSO, RBAC, white-label, on-prem LLM и др.) без переписывания
            ядра.
          </p>
        </section>

        <section>
          <h2 className="mb-3 font-serif text-[20px] leading-[28px] tracking-[-.01em]">
            Версия
          </h2>

          {isLoading && (
            <p className="text-[15px] leading-6 text-[#6b645a]">Загрузка…</p>
          )}

          {error && (
            <p className="text-[15px] leading-6 text-[#6b645a]">
              Не удалось загрузить информацию о версии. Убедитесь, что API
              доступен (в dev — бэкенд на{" "}
              <span className="font-mono text-[13px]">:8080</span>).
            </p>
          )}

          {version && (
            <dl className="flex flex-col gap-2 text-[15px] leading-6">
              <div className="flex gap-6">
                <dt className="w-32 shrink-0 text-[#6b645a]">Сборка</dt>
                <dd className="font-mono text-[#1a1a1a]">{version.version}</dd>
              </div>
              {version.module && (
                <div className="flex gap-6">
                  <dt className="w-32 shrink-0 text-[#6b645a]">Модуль</dt>
                  <dd className="font-mono text-[#1a1a1a]">{version.module}</dd>
                </div>
              )}
              {version.component && (
                <div className="flex gap-6">
                  <dt className="w-32 shrink-0 text-[#6b645a]">Компонент</dt>
                  <dd className="font-mono text-[#1a1a1a]">
                    {version.component}
                  </dd>
                </div>
              )}
              {version.commit && (
                <div className="flex gap-6">
                  <dt className="w-32 shrink-0 text-[#6b645a]">Коммит</dt>
                  <dd className="font-mono text-[#1a1a1a]">
                    {version.commit.slice(0, 8)}
                  </dd>
                </div>
              )}
              {version.build_time && (
                <div className="flex gap-6">
                  <dt className="w-32 shrink-0 text-[#6b645a]">Время сборки</dt>
                  <dd className="text-[#1a1a1a]">{version.build_time}</dd>
                </div>
              )}
              {version.license && (
                <div className="flex gap-6">
                  <dt className="w-32 shrink-0 text-[#6b645a]">Лицензия</dt>
                  <dd className="font-mono text-[#1a1a1a]">{version.license}</dd>
                </div>
              )}
            </dl>
          )}
        </section>

        <section>
          <h2 className="mb-3 font-serif text-[20px] leading-[28px] tracking-[-.01em]">
            Лицензия
          </h2>
          <p className="text-[15px] leading-6 text-[#6b645a]">
            Redpolitika CE распространяется по Business Source License (BSL)
            с Additional Use Grant. Условия и пороги — в файле LICENSE
            репозитория. Enterprise Edition лицензируется отдельно.
          </p>
        </section>
      </article>
    </div>
  );
}

export default function AboutPage() {
  return (
    <QueryClientProvider client={queryClient}>
      <AboutContent />
    </QueryClientProvider>
  );
}
