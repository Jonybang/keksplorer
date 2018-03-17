"use strict";

var ExtractTextPlugin = require('extract-text-webpack-plugin');
var extractScss = new ExtractTextPlugin('./public/assets/build/[name]');

var config = {
	name: "Frontend",
	entry: {
		'babel-polyfill': 'babel-polyfill',
		'app.css': './styles/app.scss'
	},
	output: {
		filename: './public/assets/build/[name]'
	},
    devtool: 'inline-source-map',
    module: {
        rules: [
            {
                test: /\.(ttf|woff|woff2|eot)$/,
                loader: 'file-loader',
                options: {
                    name: 'fonts/[name].[ext]'
                }
            },
            {
                test: /\.(css|scss)/,
                use: extractScss.extract({
                        fallback: "style-loader",
                        use: [ {
                            loader: "css-loader"
                        }, {
                            loader: "resolve-url-loader"
                        }, {
                            loader: "sass-loader",
                            options: {autoprefixer: true}
                        }]
                    })
            }
        ]
    },
	plugins: [
		extractScss
	]
};

module.exports = config;