/* eslint-disable @next/next/no-img-element */
export default function Logo({
  className = "",
  imgClass = "h-8",
}: {
  className?: string;
  imgClass?: string;
}) {
  return (
    <span
      className={`inline-flex items-center rounded-xl bg-white px-2.5 py-1.5 shadow-[0_6px_20px_-10px_rgba(0,0,0,.6)] ${className}`}
    >
      <img
        src="/logo.png"
        alt="HKGroup — Dược Liệu Lên Men"
        className={`${imgClass} w-auto`}
      />
    </span>
  );
}
