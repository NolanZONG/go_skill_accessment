const jwt = require("jsonwebtoken");
const { ApiError } = require("../utils");
const { env } = require("../config");

const authenticateToken = (req, res, next) => {
  // Bearer route for server-to-server callers / scripts / Postman.
  // CSRF protection is bypassed for these requests (see csrf-protection.js);
  // CSRF only matters when the browser auto-attaches cookies, which it never
  // does for a manually-set Authorization header.
  const authHeader = req.headers.authorization;
  if (authHeader?.startsWith("Bearer ")) {
    const bearerToken = authHeader.slice(7);
    return jwt.verify(bearerToken, env.JWT_ACCESS_TOKEN_SECRET, (err, user) => {
      if (err) {
        throw new ApiError(401, "Unauthorized. Invalid bearer token.");
      }
      req.user = user;
      req.isBearerAuth = true;
      next();
    });
  }

  // Cookie route for the browser frontend.
  const accessToken = req.cookies.accessToken;
  const refreshToken = req.cookies.refreshToken;

  if (!accessToken || !refreshToken) {
    throw new ApiError(401, "Unauthorized. Please provide valid tokens.");
  }

  jwt.verify(accessToken, env.JWT_ACCESS_TOKEN_SECRET, (err, user) => {
    if (err) {
      throw new ApiError(
        401,
        "Unauthorized. Please provide valid access token."
      );
    }

    jwt.verify(
      refreshToken,
      env.JWT_REFRESH_TOKEN_SECRET,
      (err, refreshToken) => {
        if (err) {
          throw new ApiError(
            401,
            "Unauthorized. Please provide valid refresh token."
          );
        }

        req.user = user;
        req.refreshToken = refreshToken;
        next();
      }
    );
  });
};

module.exports = { authenticateToken };
