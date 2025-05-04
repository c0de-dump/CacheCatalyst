self.addEventListener("install", (evt) => {
    console.log("Service worker installed");
    // Force the waiting service worker to become the active service worker.
    self.skipWaiting();
  })
  
  self.addEventListener("activate", (evt) => {
    console.log("Service worker activated");
    // Take control of all clients under this service worker's scope immediately.
    evt.waitUntil(self.clients.claim());
  })
  
  /**
   * Check if url and referrer are from the same origin.
   * CONSIDER USING self.location.origin for more reliable check.
   *
   * @param {string} requestUrl
   * @param {string} pageOrigin - e.g., self.location.origin
   * @returns
   */
  const isSameOrigin = (requestUrl, pageOrigin) => {
    try {
      const urlOrigin = new URL(requestUrl).origin;
      return urlOrigin.toLowerCase() === pageOrigin.toLowerCase();
    } catch (e) {
      // Handle invalid URLs if necessary
      return false;
    }
  }
  
  /**
   * Put response in cache.
   *
   * @param {Request} request
   * @param {Response} response
   */
  const putInCache = async (request, response) => {
    const cache = await caches.open("v2");
    await cache.put(request, response);
  };
  
  /**
   * Try-Cache, if not found try network (proxying cross-origin) and put in cache.
   *
   * @param {Request} req
   * @returns {Promise<Response>}
   */
  const cacheFirst = async (req) => {
    // First try to get the resource from the cache
    const options = { cache: "default" }; // Default: use cache if valid
    const resFromCache = await caches.match(req);
  
    if (resFromCache) {
      const etag = resFromCache.headers.get("Etag");
      // Use request URL directly as key (assuming referrer isn't part of uniqueness)
      const key = req.url;
      const cachedEtag = self.etags?.[key];
  
      if (cachedEtag) {
        if (etag == cachedEtag) {
          return resFromCache;
        } else {
          // ETag mismatch, force network reload (potentially via proxy)
          options.cache = "reload";
        }
      } else {
         // In cache, but no known ETag in self.etags, might be stale.
         // Decide whether to serve from cache or reload. For now, serve from cache.
         // Alternatively, set options.cache = "reload" here too.
         return resFromCache;
      }
    }
  
    // Not in cache or ETag mismatch, try network.
  
    let fetchRequest = req;
    const requestUrl = req.url;
    const pageOrigin = self.location.origin; // Get current service worker origin
  
    // Check if it's a cross-origin request
    if (!isSameOrigin(requestUrl, pageOrigin)) {
    //   console.log(`[Network] Cross-origin request detected for: ${requestUrl}`);
      // Construct the proxy URL pointing to your Caddy server
      const proxyUrl = `${pageOrigin}/proxy-resource?url=${encodeURIComponent(requestUrl)}`;
    //   console.log(`[Network] Rewriting to proxy: ${proxyUrl}`);
      // Create a new request object targeting the proxy
      fetchRequest = new Request(proxyUrl, {
        method: req.method, // Preserve original method (likely GET)
        headers: req.headers, // Preserve original headers if needed
        mode: 'cors', // Important for cross-origin requests via proxy
        cache: options.cache, // Carry over the cache option (e.g., 'reload')
        redirect: 'follow', // Let the browser handle redirects from the proxy
        // referrer: req.referrer, // Referrer might point to the proxy now
      });
    }
  
  
    // Next try to get the resource from the network (either directly or via proxy)
    try {
      const resFromNetwork = await fetch(fetchRequest, options); // Use fetchRequest here
  
      // Check for special header from Caddy indicating it's the initial HTML load
      // This header might now come from the proxy response if the HTML itself was proxied,
      // or directly if it was a same-origin request.
      const etagsJson = resFromNetwork.headers.get("X-Etag-Config");
      if (etagsJson != null) {
        // console.log(`[Network] Found X-Etag-Config for ${req.url}. Parsing and updating self.etags.`);
        self.etags = JSON.parse(etagsJson);
        // Don't cache the initial HTML load response itself typically
        // return resFromNetwork;
      }
  
      // If the response came from the proxy, the ETag was already handled server-side.
      // If it was a direct same-origin fetch, the ETag might be in the response.
      // We still cache the response regardless of origin.
    //   console.log(`[Network] Putting response for ${req.url} into cache.`);
      putInCache(req, resFromNetwork.clone()); // Use original req as cache key
  
      return resFromNetwork;
  
    } catch (error) {
      console.error(`[Network] Fetch error for ${fetchRequest.url}:`, error);
      // Provide a generic error response or try to return an offline fallback from cache
      const cachedFallback = await caches.match(req);
      if (cachedFallback) {
          console.warn(`[Network] Serving stale from cache due to fetch error for ${req.url}`);
          return cachedFallback;
      }
      // Generic network error response
      return new Response(`Network error proxying or fetching ${req.url}`, {
        status: 408, // Request Timeout or 500 Internal Server Error
        headers: { "Content-Type": "text/plain" },
      });
    }
  };
  
  self.addEventListener("fetch", (evt) => {
    if (evt.request.method !== "GET") {
      return;
    }
    evt.respondWith(cacheFirst(evt.request));
  });