import { useState } from "react";
import axios from "axios";
const BACKEND_UPLOAD_URL = "http://localhost:3000";

function App() {
  const [repoUrl, setRepoUrl] = useState(
    "https://github.com/nidhey27/react-counter.git"
  );
  const [uploadId, setUploadId] = useState("");
  const [uploading, setUploading] = useState(false);
  const [deployed, setDeployed] = useState(false);

  return (
    <>
      <div className="flex p-10 flex-row gap-6 items-center justify-center h-screen">
        <div className="card w-1/2 border-2 border-black-400 bg-white p-20 rounded-xl justify-start">
          <h1 className="text-2xl font-bold">Deploy your GitHub Repository</h1>
          <h2 className="text-gray-500 text-xl font-medium">
            Enter the URL of your Github Repository to deploy it
          </h2>

          <div className="mb-4 mt-8">
            <label
              className="block text-gray-700  font-bold mb-2"
              htmlFor="url"
            >
              Github Repository URL
            </label>
            <input
              value={repoUrl}
              onChange={(event) => {
                setRepoUrl(event.target.value);
              }}
              className="shadow-sm appearance-none border rounded w-full py-4 px-4 text-gray-700 leading-tight focus:outline-none focus:shadow-outline"
              id="url"
              type="text"
              placeholder="Github Repository URL"
            />

            <button
              onClick={async () => {
                setUploading(true);
                const res = await axios.post(`${BACKEND_UPLOAD_URL}/deploy`, {
                  project_url: repoUrl,
                });
                setUploadId(res.data.data);
                setUploading(false);

                const interval = setInterval(async () => {
                  const response = await axios.get(
                    `${BACKEND_UPLOAD_URL}/status?id=${res.data.data}`
                  );

                  if (response.data.status === "deployed") {
                    clearInterval(interval);
                    setDeployed(true);
                  }
                }, 5000);
              }}
              className="bg-gray-700 mt-5 w-full hover:bg-gray-900 text-white font-bold py-2 px-4 rounded focus:outline-none focus:shadow-outline"
              type="button"
            >
              {uploadId && !deployed
                ? `Deploying (${uploadId})`
                : deployed
                ? "Deployed"
                : uploading
                ? "Uploading..."
                : "Upload"}
            </button>
          </div>
        </div>

        {uploading || deployed && (
          <div className="card w-1/2 border-2 border-black-400 bg-white p-20 rounded-xl justify-start">
            <h1 className="text-2xl font-bold">Deployment Status</h1>
            {deployed && (
              <h2 className="text-gray-500 text-xl font-medium">
                Your website is successfully deployed
              </h2>
            )}
            {deployed ? (
              <>
                <div className="mb-4 mt-8">
                  <label
                    className="block text-gray-700  font-bold mb-2"
                    htmlFor="deployment-url"
                  >
                    Deployment URL
                  </label>
                  <input
                    className="shadow-sm appearance-none border rounded w-full py-4 px-4 text-gray-700 leading-tight focus:outline-none focus:shadow-outline"
                    id="deployment-url"
                    type="text"
                    placeholder="Github Repository URL"
                    value={`http://${uploadId}.nyctonid.com:3001`}
                  />

                  <a
                    href={`http://${uploadId}.nyctonid.com:3001`}
                    target="_blank"
                  >
                    <button
                      className="bg-gray-100 outline-2 mt-5 w-full hover:bg-gray-900 text-gray-500 hover:text-white font-bold py-2 px-4 rounded focus:outline-none focus:shadow-outline"
                      type="button"
                    >
                      Visit Website
                    </button>
                  </a>
                </div>
              </>
            ) : (
              <div className="flex justify-center">
                <svg
                  version="1.1"
                  id="L2"
                  xmlns="http://www.w3.org/2000/svg"
                  xmlns:xlink="http://www.w3.org/1999/xlink"
                  x="0px"
                  y="0px"
                  viewBox="0 0 100 100"
                  enable-background="new 0 0 100 100"
                  xml:space="preserve"
                >
                  <circle
                    fill="none"
                    stroke="#000"
                    stroke-width="4"
                    stroke-miterlimit="10"
                    cx="50"
                    cy="50"
                    r="48"
                  />
                  <line
                    fill="none"
                    stroke-linecap="round"
                    stroke="#fff"
                    stroke-width="4"
                    stroke-miterlimit="10"
                    x1="50"
                    y1="50"
                    x2="85"
                    y2="50.5"
                  >
                    <animateTransform
                      attributeName="transform"
                      dur="2s"
                      type="rotate"
                      from="0 50 50"
                      to="360 50 50"
                      repeatCount="indefinite"
                    />
                  </line>
                  <line
                    fill="none"
                    stroke-linecap="round"
                    stroke="#000"
                    stroke-width="4"
                    stroke-miterlimit="10"
                    x1="50"
                    y1="50"
                    x2="49.5"
                    y2="74"
                  >
                    <animateTransform
                      attributeName="transform"
                      dur="15s"
                      type="rotate"
                      from="0 50 50"
                      to="360 50 50"
                      repeatCount="indefinite"
                    />
                  </line>
                </svg>
              </div>
            )}
          </div>
        )}
      </div>
    </>
  );
}

export default App;
