import React from "react";

const ItemAvailable: React.FC = (): JSX.Element => {

  return (
    <>
    <div className="bg-green-100 text-center justify-center rounded-md p-3 flex">
    <svg
        className="stroke-2 stroke-current text-center justify-center text-green-600 h-8 w-8 mr-2 flex-shrink-0"
        viewBox="0 0 24 24"
        fill="none"
        strokeLinecap="round"
        strokeLinejoin="round"
    >
        <path d="M0 0h24v24H0z" stroke="none" />
        <circle cx="12" cy="12" r="9" />
        <path d="M9 12l2 2 4-4" />
    </svg>

    <div className="text-green-700 text-center justify-center">
        <div className="font-bold text-xl">Redeem Code available!</div>
        <div>Redirecting to home page.</div>
    </div>
</div>
    </>
  );
};

export default ItemAvailable;